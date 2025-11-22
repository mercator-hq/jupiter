package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"mercator-hq/jupiter/pkg/providers"
)

// streamReader reads Server-Sent Events (SSE) from Anthropic's streaming API.
type streamReader struct {
	provider *providers.HTTPProvider
	resp     io.ReadCloser
	scanner  *bufio.Scanner
	state    *streamState
	closed   bool
}

// newStreamReader creates a new stream reader for Anthropic's SSE stream.
func newStreamReader(ctx context.Context, provider *providers.HTTPProvider, url string, req *AnthropicRequest, headers map[string]string) (*streamReader, error) {
	// Marshal request
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Perform request
	resp, err := provider.DoRequest(ctx, "POST", url, bodyBytes, headers)
	if err != nil {
		return nil, err
	}

	// Create scanner for reading SSE lines
	scanner := bufio.NewScanner(resp.Body)

	return &streamReader{
		provider: provider,
		resp:     resp.Body,
		scanner:  scanner,
		state:    &streamState{},
		closed:   false,
	}, nil
}

// Read reads the next chunk from the stream.
// Returns nil, io.EOF when the stream ends normally.
// Returns nil, error if an error occurs.
func (s *streamReader) Read(ctx context.Context) (*providers.StreamChunk, error) {
	if s.closed {
		return nil, io.EOF
	}

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Read next SSE event
		event, err := s.readEvent()
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, &providers.StreamError{
				Provider: s.provider.GetName(),
				Message:  "failed to read stream",
				Cause:    err,
			}
		}

		if event == nil {
			continue // Skip empty events
		}

		// Transform event to chunk
		chunk, err := transformStreamChunk(event, s.state)
		if err != nil {
			return nil, &providers.ParseError{
				Provider: s.provider.GetName(),
				Cause:    err,
			}
		}

		// Some events don't produce chunks (message_start, content_block_start, etc.)
		if chunk == nil {
			continue
		}

		// Check for stream end
		if event.Type == "message_stop" {
			return nil, io.EOF
		}

		return chunk, nil
	}
}

// readEvent reads a complete SSE event.
func (s *streamReader) readEvent() (*AnthropicStreamEvent, error) {
	var eventType string
	var dataLines []string

	for s.scanner.Scan() {
		line := s.scanner.Text()

		// Empty line marks end of event
		if line == "" {
			if eventType != "" || len(dataLines) > 0 {
				break
			}
			continue
		}

		// Parse SSE field
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			dataLines = append(dataLines, data)
		}
		// Ignore other SSE fields (id, retry)
	}

	if err := s.scanner.Err(); err != nil {
		return nil, err
	}

	// No event found
	if eventType == "" && len(dataLines) == 0 {
		return nil, io.EOF
	}

	// Combine multi-line data
	var data string
	if len(dataLines) > 0 {
		data = strings.Join(dataLines, "\n")
	}

	// Parse event data
	var event AnthropicStreamEvent
	if data != "" {
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return nil, &providers.ParseError{
				Provider:    s.provider.GetName(),
				RawResponse: data,
				Cause:       fmt.Errorf("failed to parse stream event: %w", err),
			}
		}
	}

	// Set event type if parsed from SSE
	if eventType != "" && event.Type == "" {
		event.Type = eventType
	}

	return &event, nil
}

// Close closes the stream and releases resources.
func (s *streamReader) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	return s.resp.Close()
}

// readSSEData reads a complete SSE data event (handles multi-line data).
func readSSEData(scanner *bufio.Scanner) (string, error) {
	var buf bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		// Empty line marks end of event
		if line == "" {
			break
		}

		// Parse SSE field
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(data)
		}
		// Ignore other SSE fields (event, id, retry)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
