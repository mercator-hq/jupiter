package content

// ContentAnalysis contains content safety and analysis results.
// This includes PII detection, sensitive content, prompt injection, and sentiment.
type ContentAnalysis struct {
	// PIIDetection contains PII detection results.
	PIIDetection *PIIDetection

	// SensitiveContent contains sensitive content detection results.
	SensitiveContent *SensitiveContent

	// PromptInjection contains prompt injection detection results.
	PromptInjection *PromptInjection

	// Sentiment contains sentiment analysis results.
	Sentiment *Sentiment

	// Language is the detected language code (e.g., "en", "es", "fr").
	Language string

	// WordCount is the total number of words in the content.
	WordCount int

	// SentenceCount is the total number of sentences.
	SentenceCount int

	// AverageWordLength is the average word length in characters.
	AverageWordLength float64
}

// PIIDetection contains personally identifiable information detection results.
// Uses regex-based detection for common PII patterns.
type PIIDetection struct {
	// HasPII indicates whether PII was detected.
	HasPII bool

	// PIITypes lists the types of PII found (email, phone, ssn, credit_card, etc.).
	PIITypes []string

	// PIICount is the total number of PII instances found.
	PIICount int

	// Locations contains the positions of detected PII.
	Locations []PIILocation
}

// PIILocation describes the position of detected PII in the content.
type PIILocation struct {
	// Type is the PII type (email, phone, ssn, credit_card, etc.).
	Type string

	// Start is the starting character index.
	Start int

	// End is the ending character index.
	End int

	// Confidence is the detection confidence from 0.0 to 1.0.
	Confidence float64
}

// SensitiveContent contains sensitive content detection results.
// Uses keyword matching for profanity, violence, hate speech, etc.
type SensitiveContent struct {
	// HasSensitiveContent indicates whether sensitive content was detected.
	HasSensitiveContent bool

	// Categories lists the categories of sensitive content found.
	Categories []string

	// Severity indicates the severity level (low, medium, high, critical).
	Severity string

	// MatchCount is the number of sensitive keywords matched.
	MatchCount int
}

// PromptInjection contains prompt injection detection results.
// Detects jailbreak attempts and instruction override patterns.
type PromptInjection struct {
	// HasPromptInjection indicates whether prompt injection was detected.
	HasPromptInjection bool

	// InjectionType describes the type of injection (direct, indirect, jailbreak).
	InjectionType string

	// Confidence is the detection confidence from 0.0 to 1.0.
	Confidence float64

	// MatchedPatterns lists the injection patterns that matched.
	MatchedPatterns []string
}

// Sentiment contains sentiment analysis results.
// Uses rule-based analysis for MVP (no ML required).
type Sentiment struct {
	// Score is the sentiment score from -1.0 (negative) to 1.0 (positive).
	Score float64

	// Label describes the sentiment (negative, neutral, positive).
	Label string

	// Confidence is the analysis confidence from 0.0 to 1.0.
	Confidence float64
}
