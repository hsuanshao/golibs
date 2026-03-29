package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hsuanshao/golibs/ctx"
)

func TestIsValidateCloudServiceProvider(t *testing.T) {
	c := ctx.Background()
	tests := []struct {
		name     string
		csp      string
		expected bool
	}{
		{"aws lowercase", "aws", true},
		{"AWS uppercase", "AWS", true},
		{"gcp", "gcp", true},
		{"azure", "azure", true},
		{"minio", "minio", true},
		{"unknown provider", "oracle", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidateCloudServiceProvider(c, tt.csp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentTypeString(t *testing.T) {
	assert.Equal(t, "application/json", JSON.String())
	assert.Equal(t, "image/png", PNG.String())
	assert.Equal(t, "text/plain", TextPlain.String())
}

func TestIsMatchContentType(t *testing.T) {
	tests := []struct {
		name     string
		detected string
		expected ContentType
		match    bool
	}{
		// Tier 1: exact match
		{"exact match png", "image/png", PNG, true},
		{"exact match json", "application/json", JSON, true},

		// Tier 2: prefix match (charset variants)
		{"json with charset", "application/json; charset=utf-8", JSON, true},
		{"text plain with charset", "text/plain; charset=utf-8", TextPlain, true},

		// Tier 3: generic fallback - trust caller
		{"octet-stream trust caller json", "application/octet-stream", JSON, true},
		{"text/plain trust caller json", "text/plain", JSON, true},
		{"text/plain charset trust caller json", "text/plain; charset=utf-8", JSON, true},

		// Mismatch cases
		{"png detected but expected json", "image/png", JSON, false},
		{"pdf detected but expected png", "application/pdf", PNG, false},
		{"html detected but expected json", "text/html", JSON, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMatchContentType(tt.detected, tt.expected)
			assert.Equal(t, tt.match, result)
		})
	}
}
