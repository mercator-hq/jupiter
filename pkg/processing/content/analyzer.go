package content

import (
	"regexp"
	"strings"
	"sync"

	"mercator-hq/jupiter/pkg/config"
)

// Analyzer performs content analysis on text including PII detection,
// sensitive content detection, prompt injection detection, and sentiment analysis.
type Analyzer struct {
	config *config.ContentConfig

	// Compiled regex patterns for performance
	piiPatterns       map[string]*regexp.Regexp
	injectionPatterns []*regexp.Regexp

	// mu protects the analyzer for concurrent access
	mu sync.RWMutex
}

// NewAnalyzer creates a new content analyzer with the given configuration.
func NewAnalyzer(cfg *config.ContentConfig) *Analyzer {
	a := &Analyzer{
		config:      cfg,
		piiPatterns: make(map[string]*regexp.Regexp),
	}

	// Compile PII detection patterns
	a.compilePIIPatterns()

	// Compile injection detection patterns
	a.compileInjectionPatterns()

	return a
}

// AnalyzeText performs comprehensive content analysis on text.
// Returns analysis results including PII, sensitive content, prompt injection, and sentiment.
func (a *Analyzer) AnalyzeText(text string) (*ContentAnalysis, error) {
	if text == "" {
		return &ContentAnalysis{}, nil
	}

	analysis := &ContentAnalysis{}

	// Detect PII
	if a.config.PII.Enabled {
		analysis.PIIDetection = a.detectPII(text)
	}

	// Detect sensitive content
	if a.config.Sensitive.Enabled {
		analysis.SensitiveContent = a.detectSensitiveContent(text)
	}

	// Detect prompt injection
	if a.config.Injection.Enabled {
		analysis.PromptInjection = a.detectPromptInjection(text)
	}

	// Analyze sentiment
	analysis.Sentiment = a.analyzeSentiment(text)

	// Calculate text statistics
	analysis.WordCount = countWords(text)
	analysis.SentenceCount = countSentences(text)
	if analysis.WordCount > 0 {
		analysis.AverageWordLength = float64(len(text)) / float64(analysis.WordCount)
	}

	// Detect language (simplified - just check for common indicators)
	analysis.Language = detectLanguage(text)

	return analysis, nil
}

// compilePIIPatterns compiles regex patterns for PII detection.
func (a *Analyzer) compilePIIPatterns() {
	// Email pattern
	a.piiPatterns["email"] = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	// Phone pattern (international format)
	a.piiPatterns["phone"] = regexp.MustCompile(`\b(\+?1?[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)

	// SSN pattern (US Social Security Number)
	a.piiPatterns["ssn"] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Credit card pattern (basic validation)
	a.piiPatterns["credit_card"] = regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)

	// IP address pattern
	a.piiPatterns["ip_address"] = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
}

// compileInjectionPatterns compiles regex patterns for prompt injection detection.
func (a *Analyzer) compileInjectionPatterns() {
	a.injectionPatterns = make([]*regexp.Regexp, 0, len(a.config.Injection.Patterns))

	for _, pattern := range a.config.Injection.Patterns {
		// Create case-insensitive pattern
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(pattern))
		a.injectionPatterns = append(a.injectionPatterns, re)
	}
}

// detectPII detects personally identifiable information in text.
func (a *Analyzer) detectPII(text string) *PIIDetection {
	detection := &PIIDetection{
		PIITypes:  make([]string, 0),
		Locations: make([]PIILocation, 0),
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check each PII type that's enabled in config
	for _, piiType := range a.config.PII.Types {
		pattern, ok := a.piiPatterns[piiType]
		if !ok {
			continue
		}

		// Find all matches
		matches := pattern.FindAllStringIndex(text, -1)
		if len(matches) > 0 {
			detection.HasPII = true
			detection.PIITypes = append(detection.PIITypes, piiType)
			detection.PIICount += len(matches)

			// Record locations
			for _, match := range matches {
				detection.Locations = append(detection.Locations, PIILocation{
					Type:       piiType,
					Start:      match[0],
					End:        match[1],
					Confidence: 1.0, // Regex matches have high confidence
				})
			}
		}
	}

	return detection
}

// detectSensitiveContent detects sensitive content using keyword matching.
func (a *Analyzer) detectSensitiveContent(text string) *SensitiveContent {
	detection := &SensitiveContent{
		Categories: make([]string, 0),
		Severity:   "low",
	}

	textLower := strings.ToLower(text)

	// Define keyword lists for each category
	categoryKeywords := map[string][]string{
		"profanity": {
			"fuck", "shit", "damn", "ass", "bitch",
		},
		"violence": {
			"kill", "murder", "attack", "weapon", "blood", "death",
		},
		"hate_speech": {
			"hate", "racist", "discrimination",
		},
		"adult_content": {
			"sex", "porn", "nude", "explicit",
		},
	}

	totalMatches := 0

	// Check each category
	for _, category := range a.config.Sensitive.Categories {
		keywords, ok := categoryKeywords[category]
		if !ok {
			continue
		}

		categoryMatches := 0
		for _, keyword := range keywords {
			if strings.Contains(textLower, keyword) {
				categoryMatches++
			}
		}

		if categoryMatches > 0 {
			detection.HasSensitiveContent = true
			detection.Categories = append(detection.Categories, category)
			totalMatches += categoryMatches
		}
	}

	detection.MatchCount = totalMatches

	// Determine severity based on match count
	if totalMatches >= 5 {
		detection.Severity = "critical"
	} else if totalMatches >= 3 {
		detection.Severity = "high"
	} else if totalMatches >= 1 {
		detection.Severity = "medium"
	}

	return detection
}

// detectPromptInjection detects prompt injection attempts.
func (a *Analyzer) detectPromptInjection(text string) *PromptInjection {
	detection := &PromptInjection{
		MatchedPatterns: make([]string, 0),
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check against configured patterns
	for i, pattern := range a.injectionPatterns {
		if pattern.MatchString(text) {
			detection.HasPromptInjection = true
			detection.MatchedPatterns = append(detection.MatchedPatterns, a.config.Injection.Patterns[i])
		}
	}

	if detection.HasPromptInjection {
		// Determine injection type based on patterns
		if containsAny(detection.MatchedPatterns, []string{"ignore", "disregard", "forget"}) {
			detection.InjectionType = "direct"
		} else if containsAny(detection.MatchedPatterns, []string{"you are now"}) {
			detection.InjectionType = "jailbreak"
		} else {
			detection.InjectionType = "indirect"
		}

		// Set confidence based on number of matches
		if len(detection.MatchedPatterns) >= 3 {
			detection.Confidence = 0.95
		} else if len(detection.MatchedPatterns) >= 2 {
			detection.Confidence = 0.85
		} else {
			detection.Confidence = 0.75
		}
	}

	return detection
}

// analyzeSentiment performs simple rule-based sentiment analysis.
func (a *Analyzer) analyzeSentiment(text string) *Sentiment {
	sentiment := &Sentiment{
		Confidence: 0.7, // Rule-based has moderate confidence
	}

	textLower := strings.ToLower(text)

	// Positive keywords
	positiveWords := []string{
		"good", "great", "excellent", "amazing", "wonderful",
		"happy", "love", "best", "thank", "perfect",
	}

	// Negative keywords
	negativeWords := []string{
		"bad", "terrible", "awful", "horrible", "worst",
		"hate", "angry", "sad", "wrong", "fail",
	}

	positiveCount := 0
	negativeCount := 0

	// Count positive words
	for _, word := range positiveWords {
		positiveCount += strings.Count(textLower, word)
	}

	// Count negative words
	for _, word := range negativeWords {
		negativeCount += strings.Count(textLower, word)
	}

	// Calculate sentiment score (-1.0 to 1.0)
	total := positiveCount + negativeCount
	if total > 0 {
		sentiment.Score = float64(positiveCount-negativeCount) / float64(total)
	}

	// Determine label
	if sentiment.Score > 0.2 {
		sentiment.Label = "positive"
	} else if sentiment.Score < -0.2 {
		sentiment.Label = "negative"
	} else {
		sentiment.Label = "neutral"
	}

	return sentiment
}

// countWords counts the number of words in text.
func countWords(text string) int {
	if text == "" {
		return 0
	}

	words := strings.Fields(text)
	return len(words)
}

// countSentences counts the number of sentences in text.
func countSentences(text string) int {
	if text == "" {
		return 0
	}

	// Simple sentence counting based on punctuation
	count := 0
	for _, char := range text {
		if char == '.' || char == '!' || char == '?' {
			count++
		}
	}

	// If no sentence-ending punctuation, count as 1 sentence
	if count == 0 && len(text) > 0 {
		count = 1
	}

	return count
}

// detectLanguage performs simple language detection.
// For MVP, we just return "en" (English) as default.
func detectLanguage(text string) string {
	// For MVP, assume English
	// Future enhancement: use language detection library
	return "en"
}

// containsAny checks if any of the needles are in the haystack.
func containsAny(haystack []string, needles []string) bool {
	for _, h := range haystack {
		for _, n := range needles {
			if strings.Contains(strings.ToLower(h), strings.ToLower(n)) {
				return true
			}
		}
	}
	return false
}
