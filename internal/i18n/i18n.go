// Package i18n provides internationalization support for the pack service.
// It handles translation of user-facing messages and error messages.
package i18n

import (
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultLocale is the default language locale (English).
	DefaultLocale = "en"
	// AcceptLanguageHeader is the HTTP header name for language preference.
	AcceptLanguageHeader = "Accept-Language"
)

var (
	// defaultTranslator is the singleton translator instance.
	defaultTranslator *Translator
	translatorOnce    sync.Once
)

// Translator handles message translation for different locales.
type Translator struct {
	messages map[string]map[string]string
}

// NewTranslator creates a new translator with the default messages.
func NewTranslator() *Translator {
	return &Translator{
		messages: getDefaultMessages(),
	}
}

// GetTranslator returns the default singleton translator instance.
func GetTranslator() *Translator {
	translatorOnce.Do(func() {
		defaultTranslator = NewTranslator()
	})
	return defaultTranslator
}

// Translate returns the translated message for the given key and locale.
// Falls back to DefaultLocale if the locale is not found.
func (t *Translator) Translate(key, locale string) string {
	if locale == "" {
		locale = DefaultLocale
	}

	localeMessages, ok := t.messages[locale]
	if !ok {
		localeMessages = t.messages[DefaultLocale]
	}

	msg, ok := localeMessages[key]
	if !ok {
		// Fallback to default locale
		if defaultMessages := t.messages[DefaultLocale]; defaultMessages != nil {
			if fallbackMsg, exists := defaultMessages[key]; exists {
				return fallbackMsg
			}
		}
		return key
	}

	return msg
}

// GetLocale extracts the locale from the gin context.
// Checks Accept-Language header and falls back to DefaultLocale.
func GetLocale(c *gin.Context) string {
	acceptLang := c.GetHeader(AcceptLanguageHeader)
	if acceptLang == "" {
		return DefaultLocale
	}

	// Parse Accept-Language header (e.g., "en-US,en;q=0.9,pt;q=0.8")
	parts := strings.Split(acceptLang, ",")
	if len(parts) > 0 {
		lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])
		// Extract base language (e.g., "en" from "en-US")
		if idx := strings.Index(lang, "-"); idx > 0 {
			lang = lang[:idx]
		}
		// Normalize to lowercase
		lang = strings.ToLower(lang)
		// Validate it's a supported locale
		if _, ok := getDefaultMessages()[lang]; ok {
			return lang
		}
	}

	return DefaultLocale
}

// getDefaultMessages returns the default message translations.
func getDefaultMessages() map[string]map[string]string {
	return map[string]map[string]string{
		"en": {
			// Error messages
			"error.invalid_request":        "Invalid request",
			"error.invalid_request_body":    "Invalid request body",
			"error.internal_error":          "An unexpected error occurred",
			"error.unauthorized":            "Unauthorized",
			"error.invalid_credentials":     "User Not registered",
			"error.api_key_required":        "API key is required",
			"error.invalid_api_key":         "Invalid API key",
			"error.forbidden":               "Forbidden",
			"error.not_found":               "Not found",
			"error.rate_limit_exceeded":     "Too many requests, please try again later",
			"error.conflict":                "Conflict",
			"error.validation.items_ordered": "items_ordered: must be a positive integer",
			"error.invalid_token":           "Invalid or expired token",
			"error.token_required":           "Authentication token is required",

			// Success messages
			"success.pack_calculated": "Pack calculation completed successfully",
		},
		"pt": {
			// Error messages
			"error.invalid_request":        "Requisição inválida",
			"error.invalid_request_body":    "Corpo da requisição inválido",
			"error.internal_error":          "Ocorreu um erro inesperado",
			"error.unauthorized":            "Não autorizado",
			"error.invalid_credentials":     "Usuário não registrado",
			"error.api_key_required":        "Chave de API é obrigatória",
			"error.invalid_api_key":         "Chave de API inválida",
			"error.forbidden":               "Proibido",
			"error.not_found":               "Não encontrado",
			"error.rate_limit_exceeded":     "Muitas requisições, tente novamente mais tarde",
			"error.conflict":                "Conflito",
			"error.validation.items_ordered": "items_ordered: deve ser um inteiro positivo",
			"error.invalid_token":           "Token inválido ou expirado",
			"error.token_required":           "Token de autenticação é obrigatório",

			// Success messages
			"success.pack_calculated": "Cálculo de pacotes concluído com sucesso",
		},
		"nl": {
			// Error messages
			"error.invalid_request":        "Ongeldig verzoek",
			"error.invalid_request_body":    "Ongeldige aanvraag body",
			"error.internal_error":          "Er is een onverwachte fout opgetreden",
			"error.unauthorized":            "Niet geautoriseerd",
			"error.invalid_credentials":     "Gebruiker niet geregistreerd",
			"error.api_key_required":        "API-sleutel is vereist",
			"error.invalid_api_key":         "Ongeldige API-sleutel",
			"error.forbidden":               "Verboden",
			"error.not_found":               "Niet gevonden",
			"error.rate_limit_exceeded":     "Te veel verzoeken, probeer het later opnieuw",
			"error.conflict":                "Conflict",
			"error.validation.items_ordered": "items_ordered: moet een positief geheel getal zijn",
			"error.invalid_token":           "Ongeldig of verlopen token",
			"error.token_required":          "Authenticatietoken is vereist",

			// Success messages
			"success.pack_calculated": "Pakketberekening succesvol voltooid",
		},
	}
}
