package boot

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qwenode/sohot/i18n"
	"github.com/qwenode/sohot/types"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	V = types.Configure{}
)

var (
	StdoutWriter = types.NewSyncWriter(os.Stdout)
	StderrWriter = types.NewSyncWriter(os.Stderr)
)

func init() {
	log.Logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = StderrWriter
		w.NoColor = false
		w.TimeFormat = "2006-01-02 15:04:05"
	}))
}

func Load() {
	viper.SetConfigType("toml")
	viper.SetConfigName("sohot")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration file")
	}
	if err := viper.Unmarshal(&V); err != nil {
		log.Fatal().Err(err).Msg("Configuration parsing failed")
	}

	zerolog.SetGlobalLevel(zerolog.Level(V.Log.Level))

	lang := V.Log.Lang
	if lang == "" {
		lang = "en"
	}
	i18n.Init(lang)

	for i, s := range V.Watch.Exclude {
		V.Watch.Exclude[i] = strings.ToLower(strings.ReplaceAll(s, "\\", "/"))
	}
	if V.Build.Delay <= 0 {
		V.Build.Delay = 500
	}

	if V.Build.Name != "" {
		name := V.Build.Name
		if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(name), ".exe") {
			name += ".exe"
		}
		if abs, err := filepath.Abs(name); err == nil {
			name = abs
		}
		V.Build.Name = name
	}
}
