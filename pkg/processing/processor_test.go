package processing

import (
	"testing"

	"mercator-hq/jupiter/pkg/config"
)

func TestProcessor_Build(t *testing.T) {
	// Test that we can build a processor with default config
	cfg := &config.ProcessingConfig{}
	config.ApplyDefaults(&config.Config{Processing: *cfg})

	processor := NewProcessor(cfg)
	if processor == nil {
		t.Fatal("expected processor, got nil")
	}

	if processor.tokenEstimator == nil {
		t.Error("expected token estimator, got nil")
	}

	if processor.costCalculator == nil {
		t.Error("expected cost calculator, got nil")
	}

	if processor.contentAnalyzer == nil {
		t.Error("expected content analyzer, got nil")
	}

	if processor.conversationAnalyzer == nil {
		t.Error("expected conversation analyzer, got nil")
	}
}
