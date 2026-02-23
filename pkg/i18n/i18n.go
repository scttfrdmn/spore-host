// Package i18n provides internationalization support for spawn and truffle CLIs.
// It supports multiple languages with accessibility features for screen readers.
package i18n

import (
	"embed"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed active.*.toml
var translationFS embed.FS

var (
	bundle     *i18n.Bundle
	bundleOnce sync.Once

	// Global is the global localizer instance, initialized by Init()
	Global *Localizer
)

// Config configures the localizer
type Config struct {
	// Language code (e.g., "en", "es", "fr", "de", "ja")
	// If empty, will be auto-detected
	Language string

	// Verbose enables logging of translation warnings
	Verbose bool

	// AccessibilityMode enables screen reader-friendly output
	// Implies NoEmoji and provides text alternatives
	AccessibilityMode bool

	// NoEmoji disables emoji output (replaced with ASCII alternatives)
	NoEmoji bool
}

// Localizer provides translation services
type Localizer struct {
	localizer         *i18n.Localizer
	language          string
	verbose           bool
	accessibilityMode bool
	noEmoji           bool
}

// initBundle initializes the translation bundle (called once)
func initBundle() {
	bundleOnce.Do(func() {
		bundle = i18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

		// Load all embedded translations
		supportedLangs := []string{"en", "es", "fr", "de", "ja", "pt"}
		for _, lang := range supportedLangs {
			filename := fmt.Sprintf("active.%s.toml", lang)
			_, err := bundle.LoadMessageFileFS(translationFS, filename)
			if err != nil {
				// Only log error, don't fail - file might not exist yet
				if lang != "en" {
					// English is required
					log.Printf("i18n: warning: could not load %s: %v", filename, err)
				}
			}
		}
	})
}

// Init initializes the global localizer
func Init(cfg Config) error {
	initBundle()

	// Detect language if not specified
	if cfg.Language == "" {
		cfg.Language = DetectLanguage()
	}

	// Normalize language code
	cfg.Language = NormalizeLanguage(cfg.Language)

	// Accessibility mode implies no emoji
	if cfg.AccessibilityMode {
		cfg.NoEmoji = true
	}

	// Create localizer
	localizer, err := NewLocalizer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create localizer: %w", err)
	}

	Global = localizer
	return nil
}

// NewLocalizer creates a new localizer with the given configuration
func NewLocalizer(cfg Config) (*Localizer, error) {
	initBundle()

	// Accessibility mode implies no emoji
	if cfg.AccessibilityMode {
		cfg.NoEmoji = true
	}

	// Parse language tag
	langTag, err := language.Parse(cfg.Language)
	if err != nil {
		// Fall back to English if parse fails
		langTag = language.English
	}

	// Create go-i18n localizer
	goLocalizer := i18n.NewLocalizer(bundle, langTag.String())

	return &Localizer{
		localizer:         goLocalizer,
		language:          cfg.Language,
		verbose:           cfg.Verbose,
		accessibilityMode: cfg.AccessibilityMode,
		noEmoji:           cfg.NoEmoji,
	}, nil
}

// T translates a message by key
func (l *Localizer) T(key string, data ...interface{}) string {
	var templateData map[string]interface{}

	// Handle optional data parameter
	if len(data) > 0 {
		if m, ok := data[0].(map[string]interface{}); ok {
			templateData = m
		}
	}

	msg, err := l.localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateData,
	})

	if err != nil {
		// Log warning if verbose
		if l.verbose {
			log.Printf("i18n: missing translation for key=%s lang=%s: %v",
				key, l.language, err)
		}

		// Return key as fallback (better than empty string for debugging)
		if templateData != nil {
			return fmt.Sprintf("[%s]", key)
		}
		return fmt.Sprintf("[%s]", key)
	}

	return msg
}

// Tc translates a message with count (for pluralization)
func (l *Localizer) Tc(key string, count int, data ...interface{}) string {
	templateData := map[string]interface{}{
		"Count": count,
	}

	// Merge additional data if provided
	if len(data) > 0 {
		if m, ok := data[0].(map[string]interface{}); ok {
			for k, v := range m {
				templateData[k] = v
			}
		}
	}

	msg, err := l.localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateData,
		PluralCount:  count,
	})

	if err != nil {
		if l.verbose {
			log.Printf("i18n: missing translation for key=%s lang=%s: %v",
				key, l.language, err)
		}
		return fmt.Sprintf("[%s: %d]", key, count)
	}

	return msg
}

// Tf translates with formatted data (explicit map parameter)
func (l *Localizer) Tf(key string, data map[string]interface{}) string {
	return l.T(key, data)
}

// Te translates an error message and wraps the error
func (l *Localizer) Te(key string, err error, data ...interface{}) error {
	msg := l.T(key, data...)
	if err != nil {
		return fmt.Errorf("%s: %w", msg, err)
	}
	return fmt.Errorf("%s", msg)
}

// MustT translates with a warning fallback for missing keys.
// In prior versions this panicked; now it logs a warning and returns the key
// so production binaries never crash on a missing translation string.
func (l *Localizer) MustT(key string, data ...interface{}) string {
	msg := l.T(key, data...)
	if msg == fmt.Sprintf("[%s]", key) {
		fmt.Fprintf(os.Stderr, "warning: missing translation for key: %s\n", key)
		return key
	}
	return msg
}

// Language returns the current language code
func (l *Localizer) Language() string {
	return l.language
}

// AccessibilityMode returns whether accessibility mode is enabled
func (l *Localizer) AccessibilityMode() bool {
	return l.accessibilityMode
}

// NoEmoji returns whether emoji output is disabled
func (l *Localizer) NoEmoji() bool {
	return l.noEmoji
}

// SupportedLanguages returns the list of supported language codes
func SupportedLanguages() []string {
	return []string{"en", "es", "fr", "de", "ja", "pt"}
}

// IsSupported checks if a language code is supported
// Note: lang should already be normalized (lowercase, base language only)
func IsSupported(lang string) bool {
	for _, supported := range SupportedLanguages() {
		if lang == supported {
			return true
		}
	}
	return false
}
