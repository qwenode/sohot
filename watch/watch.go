package watch

import (
    "context"
    "io"
    "io/fs"
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "sohot/e"
    "strings"
    "sync"
    "syscall"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/rs/zerolog/log"
)

var (
    change      = make(chan bool, 1000)
    stopRunning = make(chan bool, 10)
    isFirstRun  = true
)

// 清理临时文件的函数
func cleanupTempFiles() {
    // 清理当前目录下的临时文件
    matches, err := filepath.Glob("*.delete_me_*")
    if err != nil {
        log.Warn().Err(err).Msg("搜索临时文件失败")
        return
    }

    for _, tempFile := range matches {
        if err := os.Remove(tempFile); err == nil {
            log.Info().Str("文件", tempFile).Msg("清理临时文件成功")
        } else {
            log.Warn().Str("文件", tempFile).Err(err).Msg("清理临时文件失败")
        }
    }
}

// 强制删除文件的函数
func forceDeleteFile(filePath string) error {
    dir := filepath.Dir(filePath)
    matches, _ := filepath.Glob(filepath.Join(dir, "*.delete_me_*"))
    for _, tempFile := range matches {
        os.Remove(tempFile)
    }
    // 方法4: 重命名文件然后尝试删除
    tempName := filePath + ".delete_me_" + time.Now().Format("20060102150405")
    if err := os.Rename(filePath, tempName); err == nil {
        return nil
    }

    // 方法2: 使用 taskkill 强制结束可能占用文件的进程
    execName := filepath.Base(filePath)
    cmd := exec.Command("taskkill", "/F", "/IM", execName)
    cmd.Run() // 忽略错误，因为进程可能不存在
    time.Sleep(time.Millisecond * 100)

    // 再次尝试删除
    if err := os.Remove(filePath); err == nil {
        log.Info().Str("文件", filePath).Msg("taskkill后删除成功")
        return nil
    }
    // 方法5: 使用 PowerShell 强制删除
    psCmd := `Remove-Item -Path "` + filePath + `" -Force -ErrorAction SilentlyContinue`
    cmd = exec.Command("powershell", "-Command", psCmd)
    if err := cmd.Run(); err == nil {
        // 检查文件是否真的被删除了
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            log.Info().Str("文件", filePath).Msg("PowerShell删除成功")
            return nil
        }
    }

    return os.Remove(filePath) // 最后还是返回原始错误
}

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
            _ = cmd.Process.Kill()

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
                log.Info().Msg("检测到重启信号")
                consume(change)
                time.Sleep(time.Second * 1)

                // 检查可执行文件是否存在
                if _, err := os.Stat(e.V.Build.Name); os.IsNotExist(err) {
                    log.Warn().Str("文件", e.V.Build.Name).Msg("可执行文件不存在，延后重启等待编译完成")
                    continue // 跳过本次重启，继续等待
                }

                log.Info().Str("文件", e.V.Build.Name).Msg("可执行文件存在，执行重启")
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
    var countdownCancel context.CancelFunc
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
        if countdownCancel != nil {
            countdownCancel()
            countdownCancel = nil
        }
        countdownMutex.Unlock()
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

        // 创建新的context用于取消倒计时
        countdownMutex.Lock()
        ctx, cancel := context.WithCancel(context.Background())
        countdownCancel = cancel
        countdownMutex.Unlock()

        // 在goroutine中处理倒计时
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    log.Error().Interface("panic", r).Msg("倒计时goroutine发生panic")
                }
            }()

            ticker := time.NewTicker(time.Second)
            defer ticker.Stop()

            secondsLeft := totalSeconds - 1

            for {
                select {
                case <-ticker.C:
                    if secondsLeft <= 0 {
                        return
                    }
                    printCountdown(secondsLeft)
                    secondsLeft--
                case <-ctx.Done():
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
            delayTimer = time.AfterFunc(time.Duration(e.V.Build.Delay)*time.Millisecond, func() {
                // 延迟到期后执行编译
                log.Info().Msg("延迟计时器到期，开始编译")
                stopCountdown()

                // 如果有正在运行的编译进程，先终止它
                if cmd != nil && cmd.Process != nil {
                    log.Info().Msg("终止当前编译")
                    cmd.Process.Kill()
                    cmd.Process.Release()
                }

                // 生成临时可执行文件名，避免影响正在运行的程序
                tempExecName := e.V.Build.Name + ".tmp_" + time.Now().Format("20060102150405")
                
                // 清理之前可能存在的临时文件
                matches, _ := filepath.Glob(e.V.Build.Name + ".tmp_*")
                for _, tempFile := range matches {
                    os.Remove(tempFile)
                }

                // 修改编译命令，先编译到临时文件
                tempCommands := make([]string, len(commands))
                copy(tempCommands, commands)
                // 找到 -o 参数的位置并替换为临时文件名
                for i, arg := range tempCommands {
                    if arg == "-o" && i+1 < len(tempCommands) {
                        tempCommands[i+1] = tempExecName
                        break
                    }
                }
                
                // 启动新的编译
                log.Info().Strs("编译命令", append([]string{"go"}, tempCommands...)).Msg("准备执行编译命令")
                cmd = exec.Command("go", tempCommands...)
                cmd.Stdout = os.Stdout
                cmd.Stderr = os.Stderr
                err := cmd.Start()
                if err != nil {
                    log.Err(err).Msg("编译启动失败")
                    cmd = nil
                    return
                }

                log.Info().Msg("启动编译")

                // 在新的 goroutine 中等待编译完成
                go func() {
                    defer func() {
                        if r := recover(); r != nil {
                            log.Error().Interface("panic", r).Msg("编译等待goroutine发生panic")
                        }
                    }()

                    err := cmd.Wait()
                    if err != nil {
                        log.Err(err).Msg("编译错误")
                        cmd = nil
                        return
                    }

                    log.Info().Msg("编译完成")

                    // 检查临时可执行文件是否存在
                    if _, err := os.Stat(tempExecName); os.IsNotExist(err) {
                        log.Warn().Str("文件", tempExecName).Msg("临时可执行文件不存在，编译可能失败")
                        cmd = nil
                        return
                    }

                    log.Info().Str("临时文件", tempExecName).Msg("编译成功，准备替换可执行文件")
                    
                    // 如果不是第一次运行，先停止旧程序
                    if !isFirstRun {
                        log.Info().Msg("停止旧程序")
                        stopRunning <- true
                        time.Sleep(time.Millisecond * 200) // 给旧进程更多时间完全退出
                    }
                    
                    // 删除旧的可执行文件并将临时文件重命名为正式文件
                    if _, err := os.Stat(e.V.Build.Name); !os.IsNotExist(err) {
                        if err := forceDeleteFile(e.V.Build.Name); err != nil {
                            log.Warn().Err(err).Str("文件", e.V.Build.Name).Msg("删除旧可执行文件失败")
                        }
                    }
                    
                    // 将临时文件重命名为正式文件
                    if err := os.Rename(tempExecName, e.V.Build.Name); err != nil {
                        log.Err(err).Str("临时文件", tempExecName).Str("目标文件", e.V.Build.Name).Msg("重命名文件失败")
                        // 清理临时文件
                        os.Remove(tempExecName)
                        cmd = nil
                        return
                    }
                    
                    log.Info().Str("文件", e.V.Build.Name).Msg("可执行文件更新成功，启动新程序")
                    
                    if isFirstRun {
                        isFirstRun = false
                    }
                    
                    Running(input)
                    cmd = nil
                }()
            })

            // 启动倒计时显示
            startCountdown(e.V.Build.Delay)

            // 清空所有待处理的变更通知
            consume(change)

        default:
            time.Sleep(time.Second)
        }
    }
}
func Watching(input e.Run) {
    // 启动时清理临时文件
    cleanupTempFiles()

    // 设置信号处理，程序退出时清理临时文件
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        log.Info().Msg("收到退出信号，清理临时文件...")
        cleanupTempFiles()
        os.Exit(0)
    }()

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
