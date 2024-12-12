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
    isFirstRun = true
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
func Running() {
    cmd := exec.Command(e.V.Build.Name)
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
        if cmd!=nil && cmd.Process != nil {
            log.Info().Msg("停止运行")
            cmd.Process.Kill()
            cmd.Process.Release()
        }
    }()
    log.Info().Msg("程序启动")
}
func Building() {
    commands := []string{
        "build",
    }
    commands = append(commands, e.V.Build.Args...)
    commands = append(commands, "-o", e.V.Build.Name, e.V.Build.Package)
    var cmd *exec.Cmd
    
    for {
        select {
        case <-change: 
            time.Sleep(time.Millisecond * time.Duration(e.V.Build.Delay))
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
                if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
                    cmd = nil
                    log.Info().Msg("编译成功")
                } else {
                    err := cmd.Wait()
                    if err != nil {
                        log.Err(err).Msg("编译错误")
                    } else {
                        log.Info().Msg("编译完成")
                        if isFirstRun {
                            isFirstRun=false
                        }else{
                            stopRunning <- true
                        }
                        Running()
                    } 
                    cmd = nil
                }
            }
            time.Sleep(time.Second)
        }
    }
}
func Watching() {
    watchDirs := map[string]bool{}
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
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal().Err(err).Send()
    }
    go func() {
        for {
            select {
            case event := <-watcher.Events:
                if isExclude(event.Name) {
                    continue
                }
                stat, err := os.Stat(event.Name)
                if err != nil {
                    continue
                }
                if stat.IsDir() {
                    continue
                }
                log.Info().Str("事件", event.Name).Send()
                change <- true
            case err2 := <-watcher.Errors:
                log.Err(err2).Msg("监控失败")
            }
        }
    }()
    for s := range watchDirs {
        watcher.Add(s)
    }
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
