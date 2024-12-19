package watch

import (
    "github.com/fsnotify/fsnotify"
    "github.com/rs/zerolog/log"
    "io"
    "io/fs"
    "os"
    "os/exec"
    "path/filepath"
    "sohot/e"
    "strings"
    "time"
)

var (
    change      = make(chan bool, 1000)
    stopRunning = make(chan bool)
    isFirstRun  = true
)

func consume(ch chan bool) {
    for {
        select {
        case <-ch:
        default:
            return
        }
    }
}

func Running(input e.Run) {
    cmd := exec.Command(e.V.Build.Name, input.Command...)
    log.Info().Strs("Run", input.Command).Msg("运行参数")
    pipe, err := cmd.StderrPipe()
    if err != nil {
        log.Err(err).Send()
        return
    }
    stdoutPipe, err := cmd.StdoutPipe()
    if err != nil {
        log.Err(err).Send()
        return
    }
    err = cmd.Start()
    if err != nil {
        log.Err(err).Send()
        return
    }
    go io.Copy(os.Stdout, stdoutPipe)
    go io.Copy(os.Stderr, pipe)
    go func() {
        <-stopRunning
        if cmd != nil && cmd.Process != nil {
            log.Info().Msg("停止运行")
            cmd.Process.Kill()
            cmd.Process.Release()
        }
    }()
    log.Info().Msg("程序启动")
}
func Building(input e.Run) {
    if input.Only {
        for {
            select {
            case <-change:
                log.Info().Msg("重启")
                consume(change)
                time.Sleep(time.Second*1)
                if isFirstRun {
                    isFirstRun = false
                } else {
                    stopRunning <- true
                }
                Running(input)
            default:
                time.Sleep(time.Second)
            }
        }
        return
    }
    commands := []string{
        "build",
    }
    commands = append(commands, e.V.Build.Command...)
    commands = append(commands, "-o", e.V.Build.Name, e.V.Build.Package)
    var cmd *exec.Cmd
    
    for {
        select {
        case <-change:
            log.Info().Msg("重启...")
            time.Sleep(time.Second)
            consume(change)
            if cmd == nil {
                cmd = exec.Command("go", commands...)
                cmd.Stdout = os.Stdout
                cmd.Stderr = os.Stderr
                err := cmd.Start()
                if err != nil {
                    log.Err(err).Msg("编译启动失败")
                } else {
                    log.Info().Msg("启动编译")
                }
            } else {
                log.Info().Msg("终止编译")
                cmd.Process.Kill()
                cmd.Process.Release()
                cmd = nil
            }
        
        default:
            if cmd != nil {
                err := cmd.Wait()
                if err != nil {
                    log.Err(err).Msg("编译错误")
                } else {
                    log.Info().Msg("编译完成")
                    if isFirstRun {
                        isFirstRun = false
                    } else {
                        stopRunning <- true
                    }
                    Running(input)
                }
                cmd = nil
            }
            time.Sleep(time.Second)
        }
    }
}
func Watching(input e.Run) {
    watchDirs := map[string]bool{}
    if input.Only {
        watchDirs[filepath.Dir(e.V.Build.Name)] = true
    } else {
        for _, s := range e.V.Watch.Include {
            filepath.WalkDir(
                s, func(path string, d fs.DirEntry, err error) error {
                    if !d.IsDir() {
                        return nil
                    }
                    if isExclude(path) {
                        return nil
                    }
                    watchDirs[path] = true
                    return nil
                },
            )
        }
    }
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal().Err(err).Send()
    }
    go func() {
        for {
            select {
            case event := <-watcher.Events:
                if !input.Only && isExclude(event.Name) {
                    continue
                }
                stat, err := os.Stat(event.Name)
                if err != nil {
                    continue
                }
                if stat.IsDir() {
                    continue
                }
                // log.Info().Str("事件", event.Name).Send()
                change <- true
            case err2 := <-watcher.Errors:
                log.Err(err2).Msg("监控失败")
            }
        }
    }()
    for s := range watchDirs {
        watcher.Add(s)
    }
    // 20241218 先触发一个 by Node
    change <- true
}
func isExclude(path string) bool {
    path = strings.ReplaceAll(strings.ToLower(path), "\\", "/")
    for _, s := range e.V.Watch.Exclude {
        if strings.Contains(path, s) {
            return true
        }
    }
    return false
}
