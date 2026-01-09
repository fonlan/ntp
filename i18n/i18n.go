// Package i18n provides internationalization support for the application
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Translator holds the translation data for a specific locale
type Translator struct {
	locale  string
	messages map[string]interface{}
	mu      sync.RWMutex
}

var (
	translators = make(map[string]*Translator)
	mu          sync.RWMutex
	defaultLocale = "en"
	supportedLocales = []string{"en", "zh"}
)

// LoadTranslations loads translation files for all supported locales
func LoadTranslations(dir string) error {
	mu.Lock()
	defer mu.Unlock()

	for _, locale := range supportedLocales {
		filePath := filepath.Join(dir, locale+".json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read translation file %s: %w", filePath, err)
		}

		var messages map[string]interface{}
		if err := json.Unmarshal(data, &messages); err != nil {
			return fmt.Errorf("failed to parse translation file %s: %w", filePath, err)
		}

		translators[locale] = &Translator{
			locale:  locale,
			messages: messages,
		}
	}

	return nil
}

// GetTranslator returns a translator for the given locale
// If the locale is not supported, it returns the default locale translator
func GetTranslator(locale string) *Translator {
	mu.RLock()
	defer mu.RUnlock()

	// Normalize locale (e.g., "zh-CN" -> "zh")
	if idx := strings.Index(locale, "-"); idx != -1 {
		locale = locale[:idx]
	}

	if translator, ok := translators[locale]; ok {
		return translator
	}

	return translators[defaultLocale]
}

// T returns the translation for the given key
// The key format is "category.key" (e.g., "bookmark.listFailed")
func (t *Translator) T(key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	parts := strings.Split(key, ".")
	var current interface{} = t.messages

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return key // Return key if translation not found
			}
		} else {
			return key
		}
	}

	if str, ok := current.(string); ok {
		return str
	}

	return key
}

// Tf returns the formatted translation for the given key with format arguments
func (t *Translator) Tf(key string, args ...interface{}) string {
	msg := t.T(key)
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// SetDefaultLocale sets the default locale
func SetDefaultLocale(locale string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := translators[locale]; !ok {
		return fmt.Errorf("unsupported locale: %s", locale)
	}

	defaultLocale = locale
	return nil
}

// GetDefaultLocale returns the default locale
func GetDefaultLocale() string {
	return defaultLocale
}

// GetSupportedLocales returns the list of supported locales
func GetSupportedLocales() []string {
	return supportedLocales
}
