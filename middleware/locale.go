// Package middleware provides HTTP middleware functions
package middleware

import (
	"context"
	"net/http"
	"strings"

	"ntp/i18n"
)

// LocaleMiddleware adds locale information to the request context
func LocaleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get locale from Accept-Language header
		locale := getLocaleFromRequest(r)

		// Store translator in request context
		translator := i18n.GetTranslator(locale)
		r = r.WithContext(contextWithTranslator(r.Context(), translator))

		next.ServeHTTP(w, r)
	})
}

// getLocaleFromRequest extracts the locale from the Accept-Language header
func getLocaleFromRequest(r *http.Request) string {
	acceptLanguage := r.Header.Get("Accept-Language")
	if acceptLanguage == "" {
		return i18n.GetDefaultLocale()
	}

	// Parse Accept-Language header (e.g., "zh-CN,zh;q=0.9,en;q=0.8")
	parts := strings.Split(acceptLanguage, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, ";"); idx != -1 {
			part = part[:idx]
		}

		// Normalize locale (e.g., "zh-CN" -> "zh")
		if idx := strings.Index(part, "-"); idx != -1 {
			part = part[:idx]
		}

		// Check if this locale is supported
		for _, supported := range i18n.GetSupportedLocales() {
			if strings.EqualFold(part, supported) {
				return supported
			}
		}
	}

	return i18n.GetDefaultLocale()
}

// Context key type for storing translator in context
type contextKey string

const translatorKey contextKey = "translator"

// contextWithTranslator returns a new context with the translator
func contextWithTranslator(ctx context.Context, translator *i18n.Translator) context.Context {
	return context.WithValue(ctx, translatorKey, translator)
}

// TranslatorFromContext extracts the translator from the context
func TranslatorFromContext(ctx context.Context) *i18n.Translator {
	if translator, ok := ctx.Value(translatorKey).(*i18n.Translator); ok {
		return translator
	}
	return i18n.GetTranslator(i18n.GetDefaultLocale())
}
