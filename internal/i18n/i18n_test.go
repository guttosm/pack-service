//go:build !integration

package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetTranslator(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T)
	}{
		{
			name: "returns singleton translator instance",
			validate: func(t *testing.T) {
				translator1 := GetTranslator()
				translator2 := GetTranslator()
				assert.NotNil(t, translator1)
				assert.NotNil(t, translator2)
				assert.Equal(t, translator1, translator2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}

func TestTranslator_Translate(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		key      string
		locale   string
		expected string
	}{
		{
			name:     "english message",
			key:      "error.invalid_request",
			locale:   "en",
			expected: "Invalid request",
		},
		{
			name:     "portuguese message",
			key:      "error.invalid_request",
			locale:   "pt",
			expected: "Requisição inválida",
		},
		{
			name:     "dutch message",
			key:      "error.invalid_request",
			locale:   "nl",
			expected: "Ongeldig verzoek",
		},
		{
			name:     "empty locale defaults to english",
			key:      "error.invalid_request",
			locale:   "",
			expected: "Invalid request",
		},
		{
			name:     "unsupported locale falls back to english",
			key:      "error.invalid_request",
			locale:   "fr",
			expected: "Invalid request",
		},
		{
			name:     "unknown key returns key",
			key:      "unknown.key",
			locale:   "en",
			expected: "unknown.key",
		},
		{
			name:     "unknown key in unsupported locale falls back",
			key:      "unknown.key",
			locale:   "fr",
			expected: "unknown.key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.Translate(tt.key, tt.locale)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		acceptLanguage string
		expected       string
	}{
		{
			name:           "no header returns default",
			acceptLanguage: "",
			expected:       DefaultLocale,
		},
		{
			name:           "english header",
			acceptLanguage: "en",
			expected:       "en",
		},
		{
			name:           "portuguese header",
			acceptLanguage: "pt",
			expected:       "pt",
		},
		{
			name:           "dutch header",
			acceptLanguage: "nl",
			expected:       "nl",
		},
		{
			name:           "full locale with region",
			acceptLanguage: "en-US",
			expected:       "en",
		},
		{
			name:           "multiple languages",
			acceptLanguage: "en-US,en;q=0.9,pt;q=0.8",
			expected:       "en",
		},
		{
			name:           "unsupported language defaults",
			acceptLanguage: "fr",
			expected:       DefaultLocale,
		},
		{
			name:           "case insensitive",
			acceptLanguage: "EN",
			expected:       "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set(AcceptLanguageHeader, tt.acceptLanguage)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			result := GetLocale(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}
