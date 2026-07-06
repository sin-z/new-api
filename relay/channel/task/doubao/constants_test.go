package doubao

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVideoInputRatioUsesSeedance20USDPriceTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		modelName  string
		resolution string
		hasVideo   bool
		wantRatio  float64
		wantOK     bool
	}{
		{
			name:      "standard 480p without video uses base price",
			modelName: "doubao-seedance-2-0-260128",
			wantRatio: 1.0,
			wantOK:    true,
		},
		{
			name:      "standard 720p with video uses video input price",
			modelName: "doubao-seedance-2-0-260128",
			hasVideo:  true,
			wantRatio: 4.3 / 7.0,
			wantOK:    true,
		},
		{
			name:       "standard 1080p without video uses 1080p price",
			modelName:  "doubao-seedance-2-0-260128",
			resolution: "1080p",
			wantRatio:  7.7 / 7.0,
			wantOK:     true,
		},
		{
			name:       "standard 1080p with video uses 1080p video input price",
			modelName:  "doubao-seedance-2-0-260128",
			resolution: "1080p",
			hasVideo:   true,
			wantRatio:  4.7 / 7.0,
			wantOK:     true,
		},
		{
			name:       "standard 4k without video uses 4k price",
			modelName:  "doubao-seedance-2-0-260128",
			resolution: "4k",
			wantRatio:  4.0 / 7.0,
			wantOK:     true,
		},
		{
			name:       "standard 4k with video uses 4k video input price",
			modelName:  "doubao-seedance-2-0-260128",
			resolution: "4k",
			hasVideo:   true,
			wantRatio:  2.4 / 7.0,
			wantOK:     true,
		},
		{
			name:      "fast without video uses base price",
			modelName: "doubao-seedance-2-0-fast-260128",
			wantRatio: 1.0,
			wantOK:    true,
		},
		{
			name:      "fast with video uses fast video input price",
			modelName: "doubao-seedance-2-0-fast-260128",
			hasVideo:  true,
			wantRatio: 3.3 / 5.6,
			wantOK:    true,
		},
		{
			name:       "fast 1080p falls back to base ratio because upstream does not support it",
			modelName:  "doubao-seedance-2-0-fast-260128",
			resolution: "1080p",
			wantRatio:  1.0,
			wantOK:     true,
		},
		{
			name:       "unknown model has no price table",
			modelName:  "unknown-model",
			resolution: "1080p",
			wantRatio:  0,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotRatio, gotOK := GetVideoInputRatio(tt.modelName, tt.resolution, tt.hasVideo)

			require.Equal(t, tt.wantOK, gotOK)
			assert.InDelta(t, tt.wantRatio, gotRatio, 0.0000001)
		})
	}
}
