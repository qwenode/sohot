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
    "sync"
    "time"
)

var (
    change      = make(chan bool, 1000)
    stopRunning = make(chan bool, 10)
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
    consume(stopRunning)
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
            err := cmd.Process.Kill()
            if err != nil {
                log.Warn().Err(err).Msg("终止进程时出现警告")
            }
            // 等待进程完全退出
            cmd.Wait()
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
                time.Sleep(time.Second * 1)
                if isFirstRun {
                    isFirstRun = false
                } else {
                    stopRunning <- true
                    time.Sleep(time.Millisecond * 100) // 给旧进程一点时间完全退出
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

    // 用于延迟编译的计时器
    var delayTimer *time.Timer
    var delayActive bool
    var countdownTicker *time.Ticker
    var countdownDone chan bool
    var countdownMutex sync.Mutex

    // 打印倒计时的函数
    printCountdown := func(remainingSeconds int) {
        if remainingSeconds < 0 {
            remainingSeconds = 0
        }
        log.Info().Int("剩余秒数", remainingSeconds).Msg("编译倒计时")
    }

    // 停止倒计时的函数
    stopCountdown := func() {
        countdownMutex.Lock()
        defer countdownMutex.Unlock()
        
        if countdownTicker != nil {
            countdownTicker.Stop()
            countdownTicker = nil
        }
        
        // 安全地关闭通道
        if countdownDone != nil {
            select {
            case <-countdownDone:
                // 通道已经关闭
            default:
                close(countdownDone)
            }
            countdownDone = nil
        }
    }

    // 启动倒计时的函数
    startCountdown := func(delayMs int) {
        // 先停止之前的倒计时
        stopCountdown()

        // 计算总秒数
        totalSeconds := delayMs / 1000
        if delayMs%1000 > 0 {
            totalSeconds++
        }

        // 首次立即显示倒计时
        printCountdown(totalSeconds)

        // 如果延迟时间太短，不启动ticker
        if totalSeconds <= 1 {
            return
        }

        // 在锁保护下创建新的倒计时通道和ticker
        countdownMutex.Lock()
        countdownDone = make(chan bool)
        countdownTicker = time.NewTicker(time.Second)
        localTicker := countdownTicker
        localDone := countdownDone
        countdownMutex.Unlock()

        // 在goroutine中处理倒计时
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    log.Error().Interface("panic", r).Msg("倒计时goroutine发生panic")
                }
            }()
            
            secondsLeft := totalSeconds - 1

            for {
                select {
                case <-localTicker.C:
                    if secondsLeft <= 0 {
                        return
                    }
                    printCountdown(secondsLeft)
                    secondsLeft--
                case <-localDone:
                    return
                }
            }
        }()
    }

    for {
        select {
        case <-change:
            // 收到文件变更通知
            log.Info().Msg("检测到文件变更...")

            // 如果已经有一个延迟计时器在运行，停止它并重新计算延迟
            if delayTimer != nil {
                delayTimer.Stop()
                log.Info().Msg("重置延迟计时器")
            }

            // 停止当前的倒计时
            stopCountdown()

            // 设置新的延迟计时器
            delayActive = true
            delayTimer = time.AfterFunc(time.Duration(e.V.Build.Delay)*time.Millisecond, func() {
                // 延迟到期后执行编译
                delayActive = false
                stopCountdown()

                // 如果有正在运行的编译进程，先终止它
                if cmd != nil && cmd.Process != nil {
                    log.Info().Msg("终止当前编译")
                    cmd.Process.Kill()
                    cmd.Process.Release()
                    cmd = nil
                }
                
                // 先停止正在运行的程序，确保可执行文件不被锁定
                stopRunning <- true
                time.Sleep(time.Millisecond * 100) // 给进程一点时间来完全退出
                
                // 尝试删除旧的可执行文件，如果删除失败则重试几次
                for i := 0; i < 5; i++ {
                    _, err3 := os.Stat(e.V.Build.Name)
                    if os.IsNotExist(err3) {
                        break // 文件不存在，无需删除
                    }
                    
                    err3 = os.Remove(e.V.Build.Name)
                    if err3 == nil {
                        log.Info().Msg("成功删除旧的可执行文件")
                        break // 删除成功
                    }
                    
                    log.Warn().Err(err3).Int("重试次数", i+1).Msg("删除可执行文件失败，正在重试")
                    time.Sleep(time.Millisecond * 200) // 等待一段时间后重试
                }
                // 启动新的编译
                cmd = exec.Command("go", commands...)
                cmd.Stdout = os.Stdout
                cmd.Stderr = os.Stderr
                err := cmd.Start()
                if err != nil {
                    log.Err(err).Msg("编译启动失败")
                } else {
                    log.Info().Msg("启动编译")
                }
            })

            // 启动倒计时显示
            startCountdown(e.V.Build.Delay)

            // 清空所有待处理的变更通知
            consume(change)

        default:
            // 如果没有活跃的延迟计时器，并且有编译进程在运行，检查其状态
            if !delayActive && cmd != nil {
                err := cmd.Wait()
                if err != nil {
                    log.Err(err).Msg("编译错误")
                } else {
                    log.Info().Msg("编译完成")
                    if isFirstRun {
                        isFirstRun = false
                    } else {
                        stopRunning <- true
                        time.Sleep(time.Millisecond * 100) // 给旧进程一点时间完全退出
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
