package e

import (
    "io"
    "os"
    "strings"
    "sync"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/spf13/viper"
)

var (
    V = Configure{}
)

// SyncWriter 是一个线程安全的 writer，用于防止并发写入时输出混乱
type SyncWriter struct {
    mu sync.Mutex
    w  io.Writer
}

func (sw *SyncWriter) Write(p []byte) (n int, err error) {
    sw.mu.Lock()
    defer sw.mu.Unlock()
    return sw.w.Write(p)
}

// 全局同步 writer，供日志和程序输出共用
var (
    StdoutWriter = &SyncWriter{w: os.Stdout}
    StderrWriter = &SyncWriter{w: os.Stderr}
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
