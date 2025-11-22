package tracing

import (
	"testing"
)

// TestCreateSampler tests sampler creation
func TestCreateSampler(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		ratio    float64
		wantErr  bool
	}{
		{
			name:     "always sampler",
			strategy: SamplerAlways,
			ratio:    0.0,
			wantErr:  false,
		},
		{
			name:     "never sampler",
			strategy: SamplerNever,
			ratio:    0.0,
			wantErr:  false,
		},
		{
			name:     "ratio sampler - 0%",
			strategy: SamplerRatio,
			ratio:    0.0,
			wantErr:  false,
		},
		{
			name:     "ratio sampler - 50%",
			strategy: SamplerRatio,
			ratio:    0.5,
			wantErr:  false,
		},
		{
			name:     "ratio sampler - 100%",
			strategy: SamplerRatio,
			ratio:    1.0,
			wantErr:  false,
		},
		{
			name:     "ratio sampler - invalid negative",
			strategy: SamplerRatio,
			ratio:    -0.1,
			wantErr:  true,
		},
		{
			name:     "ratio sampler - invalid > 1",
			strategy: SamplerRatio,
			ratio:    1.5,
			wantErr:  true,
		},
		{
			name:     "unknown strategy",
			strategy: "unknown",
			ratio:    0.5,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler, err := createSampler(tt.strategy, tt.ratio)
			if (err != nil) != tt.wantErr {
				t.Errorf("createSampler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && sampler == nil {
				t.Error("createSampler() returned nil sampler without error")
			}
		})
	}
}

// TestValidateSamplingConfig tests sampling config validation
func TestValidateSamplingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  SamplingConfig
		wantErr bool
	}{
		{
			name: "valid always",
			config: SamplingConfig{
				Strategy: SamplerAlways,
				Ratio:    0.0,
			},
			wantErr: false,
		},
		{
			name: "valid never",
			config: SamplingConfig{
				Strategy: SamplerNever,
				Ratio:    0.0,
			},
			wantErr: false,
		},
		{
			name: "valid ratio",
			config: SamplingConfig{
				Strategy: SamplerRatio,
				Ratio:    0.1,
			},
			wantErr: false,
		},
		{
			name: "invalid strategy",
			config: SamplingConfig{
				Strategy: "invalid",
				Ratio:    0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid ratio - negative",
			config: SamplingConfig{
				Strategy: SamplerRatio,
				Ratio:    -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid ratio - too high",
			config: SamplingConfig{
				Strategy: SamplerRatio,
				Ratio:    1.5,
			},
			wantErr: true,
		},
		{
			name: "ratio strategy with ratio 0",
			config: SamplingConfig{
				Strategy: SamplerRatio,
				Ratio:    0.0,
			},
			wantErr: false,
		},
		{
			name: "ratio strategy with ratio 1",
			config: SamplingConfig{
				Strategy: SamplerRatio,
				Ratio:    1.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSamplingConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSamplingConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSamplerConstants tests sampler constant values
func TestSamplerConstants(t *testing.T) {
	// Verify constants have expected values
	if SamplerAlways != "always" {
		t.Errorf("SamplerAlways = %q, want %q", SamplerAlways, "always")
	}
	if SamplerNever != "never" {
		t.Errorf("SamplerNever = %q, want %q", SamplerNever, "never")
	}
	if SamplerRatio != "ratio" {
		t.Errorf("SamplerRatio = %q, want %q", SamplerRatio, "ratio")
	}
}
