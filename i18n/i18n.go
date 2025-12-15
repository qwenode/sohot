package i18n

import (
	"embed"
	"strings"

	"github.com/spf13/viper"
)

//go:embed messages.en.toml messages.zh.toml
var messagesFS embed.FS

var messages map[string]string

// Init initializes the i18n system with the specified language
func Init(lang string) {
	lang = strings.ToLower(lang)
	if lang != "zh" && lang != "en" {
		lang = "en"
	}

	filename := "messages." + lang + ".toml"
	data, err := messagesFS.ReadFile(filename)
	if err != nil {
		// Fallback to English
		data, _ = messagesFS.ReadFile("messages.en.toml")
	}

	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(strings.NewReader(string(data))); err != nil {
		messages = make(map[string]string)
		return
	}

	messages = make(map[string]string)
	for _, section := range []string{"boot", "main", "watch"} {
		sub := v.Sub(section)
		if sub == nil {
			continue
		}
		for key, val := range sub.AllSettings() {
			if s, ok := val.(string); ok {
				messages[section+"."+key] = s
			}
		}
	}
}

// T returns the translated message for the given key
func T(key string) string {
	if msg, ok := messages[key]; ok {
		return msg
	}
	return key
}
