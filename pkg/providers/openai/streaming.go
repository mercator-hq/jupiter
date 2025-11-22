package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"mercator-hq/jupiter/pkg/providers"
)

// streamReader reads Server-Sent Events (SSE) from OpenAI's streaming API.
type streamReader struct {
	provider *providers.HTTPProvider
	resp     io.ReadCloser
	scanner  *bufio.Scanner
	closed   bool
}

// newStreamReader creates a new stream reader for OpenAI's SSE stream.
func newStreamReader(ctx context.Context, provider *providers.HTTPProvider, url string, req *OpenAIRequest, headers map[string]string) (*streamReader, error) {
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

		// Read next line
		if !s.scanner.Scan() {
			// Check for error
			if err := s.scanner.Err(); err != nil {
				return nil, &providers.StreamError{
					Provider: s.provider.GetName(),
					Message:  "failed to read stream",
					Cause:    err,
				}
			}
			// End of stream
			return nil, io.EOF
		}

		line := s.scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE line
		if !strings.HasPrefix(line, "data: ") {
			// Skip non-data lines (comments, event types, etc.)
			continue
		}

		// Extract data
		data := strings.TrimPrefix(line, "data: ")

		// Check for stream termination
		if data == "[DONE]" {
			return nil, io.EOF
		}

		// Parse JSON chunk
		var openaiChunk OpenAIStreamResponse
		if err := json.Unmarshal([]byte(data), &openaiChunk); err != nil {
			return nil, &providers.ParseError{
				Provider:    s.provider.GetName(),
				RawResponse: data,
				Cause:       fmt.Errorf("failed to parse stream chunk: %w", err),
			}
		}

		// Transform to provider-agnostic format
		chunk, err := transformStreamChunk(&openaiChunk)
		if err != nil {
			return nil, &providers.ParseError{
				Provider: s.provider.GetName(),
				Cause:    err,
			}
		}

		return chunk, nil
	}
}

// Close closes the stream and releases resources.
func (s *streamReader) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	return s.resp.Close()
}
