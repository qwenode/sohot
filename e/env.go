package e

import (
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/spf13/viper"
    "os"
    "strings"
)

var (
    V = Configure{}
)

type (
    Configure struct {
        Log   Log            `mapstructure:"log"`
        Watch Watch          `mapstructure:"watch"`
        Build Build          `mapstructure:"build"`
        Run   map[string]Run `mapstructure:"run"`
    }
    Run struct {
        Command []string `mapstructure:"command"`
        Only    bool     `mapstructure:"only"`
    }
    Build struct {
        Delay   int      `mapstructure:"delay"`
        Name    string   `mapstructure:"name"`
        Package string   `mapstructure:"package"`
        Command []string `mapstructure:"command"`
    }
    Watch struct {
        Include []string `mapstructure:"include"`
        Exclude []string `mapstructure:"exclude"`
    }
    Log struct {
        Level int `mapstructure:"level"`
    }
)

func init() {
    log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false, TimeFormat: "2006-01-02T15:04:05Z"})
    viper.SetConfigType("toml")
    viper.SetConfigName("sohot")
    viper.AddConfigPath(".")
    err := viper.ReadInConfig()
    if err != nil {
        log.Fatal().Err(err).Msg("配置文件错误")
    }
    viper.Unmarshal(&V)
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
