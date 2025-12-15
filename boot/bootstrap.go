package boot

import (
	"os"
	"strings"

	"github.com/qwenode/sohot/types"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	V = types.Configure{}
)

// 全局同步 writer，供日志和程序输出共用
var (
	StdoutWriter = types.NewSyncWriter(os.Stdout)
	StderrWriter = types.NewSyncWriter(os.Stderr)
)

func init() {
	log.Logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = StderrWriter
		w.NoColor = false
		w.TimeFormat = "2006-01-02T15:04:05Z"
	}))
	viper.SetConfigType("toml")
	viper.SetConfigName("sohot")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Configuration file error")
	}
	if err := viper.Unmarshal(&V); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse configuration")
	}
	for i, s := range V.Watch.Exclude {
		V.Watch.Exclude[i] = strings.ToLower(s)
	}
	V.Watch.Exclude = append(V.Watch.Exclude, ".idea")
	V.Watch.Exclude = append(V.Watch.Exclude, ".git")
	V.Watch.Exclude = append(V.Watch.Exclude, ".exe")
	if V.Build.Delay <= 0 {
		V.Build.Delay = 500
	}
}
