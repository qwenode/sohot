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

// Function to clean up temporary files
func cleanupTempFiles() {
    // Clean temporary files in current directory
    matches, err := filepath.Glob("*.delete_me_*")
    if err != nil {
        log.Warn().Err(err).Msg("Failed to search temporary files")
        return
    }

    for _, tempFile := range matches {
        if err := os.Remove(tempFile); err == nil {
            log.Info().Str("file", tempFile).Msg("Cleaned temporary file successfully")
        } else {
            log.Warn().Str("file", tempFile).Err(err).Msg("Failed to clean temporary file")
        }
    }
}

// Function to force delete files
func forceDeleteFile(filePath string) error {
    dir := filepath.Dir(filePath)
    matches, _ := filepath.Glob(filepath.Join(dir, "*.delete_me_*"))
    for _, tempFile := range matches {
        os.Remove(tempFile)
    }
    tempName := filePath + ".delete_me_" + time.Now().Format("20060102150405")
    if err := os.Rename(filePath, tempName); err == nil {
        return nil
    }

    return os.Remove(filePath) // Finally return the original error
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
    log.Info().Strs("Run", input.Command).Msg("Run arguments")
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
            log.Info().Msg("Stopping execution")
            _ = cmd.Process.Kill()

            // Wait for process to completely exit
            cmd.Wait()
            cmd.Process.Release()
        }
    }()
    log.Info().Msg("Program started")
}
func Building(input e.Run) {
    if input.Only {
        for {
            select {
            case <-change:
                log.Info().Msg("Restart signal detected")
                consume(change)
                time.Sleep(time.Second * 1)

                // Check if executable file exists
                if _, err := os.Stat(e.V.Build.Name); os.IsNotExist(err) {
                    log.Warn().Str("file", e.V.Build.Name).Msg("Executable file not found, delaying restart to wait for compilation")
                    continue // Skip this restart, continue waiting
                }

                log.Info().Str("file", e.V.Build.Name).Msg("Executable file exists, performing restart")
                if isFirstRun {
                    isFirstRun = false
                } else {
                    stopRunning <- true
                    time.Sleep(time.Millisecond * 100) // Give old process some time to completely exit
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

    // Timer for delayed compilation
    var delayTimer *time.Timer
    var countdownCancel context.CancelFunc
    var countdownMutex sync.Mutex

    // Function to print countdown
    printCountdown := func(remainingSeconds int) {
        if remainingSeconds < 0 {
            remainingSeconds = 0
        }
        log.Info().Int("seconds_remaining", remainingSeconds).Msg("Compilation countdown")
    }

    // Function to stop countdown
    stopCountdown := func() {
        countdownMutex.Lock()
        if countdownCancel != nil {
            countdownCancel()
            countdownCancel = nil
        }
        countdownMutex.Unlock()
    }

    // Function to start countdown
    startCountdown := func(delayMs int) {
        // First stop previous countdown
        stopCountdown()

        // Calculate total seconds
        totalSeconds := delayMs / 1000
        if delayMs%1000 > 0 {
            totalSeconds++
        }

        // Show countdown immediately for the first time
        printCountdown(totalSeconds)

        // If delay time is too short, don't start ticker
        if totalSeconds <= 1 {
            return
        }

        // Create new context for canceling countdown
        countdownMutex.Lock()
        ctx, cancel := context.WithCancel(context.Background())
        countdownCancel = cancel
        countdownMutex.Unlock()

        // Handle countdown in goroutine
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    log.Error().Interface("panic", r).Msg("Countdown goroutine panic occurred")
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
            // Received file change notification
            log.Info().Msg("File change detected...")

            // If there's already a delay timer running, stop it and recalculate delay
            if delayTimer != nil {
                delayTimer.Stop()
                log.Info().Msg("Resetting delay timer")
            }

            // Stop current countdown
            stopCountdown()

            // Set new delay timer
            delayTimer = time.AfterFunc(time.Duration(e.V.Build.Delay)*time.Millisecond, func() {
                // Execute compilation after delay expires
                log.Info().Msg("Delay timer expired, starting compilation")
                stopCountdown()

                // If there's a running compilation process, terminate it first
                if cmd != nil && cmd.Process != nil {
                    log.Info().Msg("Terminating current compilation")
                    cmd.Process.Kill()
                    cmd.Process.Release()
                }

                // Generate temporary executable filename to avoid affecting running program
                tempExecName := e.V.Build.Name + ".tmp_" + time.Now().Format("20060102150405")

                // Clean up previously existing temporary files
                matches, _ := filepath.Glob(e.V.Build.Name + ".tmp_*")
                for _, tempFile := range matches {
                    os.Remove(tempFile)
                }

                // Modify compilation command to compile to temporary file first
                tempCommands := make([]string, len(commands))
                copy(tempCommands, commands)
                // Find the -o parameter position and replace with temporary filename
                for i, arg := range tempCommands {
                    if arg == "-o" && i+1 < len(tempCommands) {
                        tempCommands[i+1] = tempExecName
                        break
                    }
                }

                // Start new compilation
                log.Info().Strs("build_command", append([]string{"go"}, tempCommands...)).Msg("Preparing to execute build command")
                cmd = exec.Command("go", tempCommands...)
                cmd.Stdout = os.Stdout
                cmd.Stderr = os.Stderr
                err := cmd.Start()
                if err != nil {
                    log.Err(err).Msg("Failed to start compilation")
                    cmd = nil
                    return
                }

                log.Info().Msg("Starting compilation")

                // Wait for compilation to complete in new goroutine
                go func() {
                    defer func() {
                        if r := recover(); r != nil {
                            log.Error().Interface("panic", r).Msg("Compilation wait goroutine panic occurred")
                        }
                    }()

                    err := cmd.Wait()
                    if err != nil {
                        log.Err(err).Msg("Compilation error")
                        cmd = nil
                        return
                    }

                    log.Info().Msg("Compilation completed")

                    // Check if temporary executable file exists
                    if _, err := os.Stat(tempExecName); os.IsNotExist(err) {
                        log.Warn().Str("file", tempExecName).Msg("Temporary executable file not found, compilation may have failed")
                        cmd = nil
                        return
                    }

                    log.Info().Str("temp_file", tempExecName).Msg("Compilation successful, preparing to replace executable file")

                    // If not first run, stop old program first
                    if !isFirstRun {
                        log.Info().Msg("Stopping old program")
                        stopRunning <- true
                        time.Sleep(time.Millisecond * 200) // Give old process more time to completely exit
                    }

                    // Delete old executable file and rename temporary file to official file
                    if _, err := os.Stat(e.V.Build.Name); !os.IsNotExist(err) {
                        if err := forceDeleteFile(e.V.Build.Name); err != nil {
                            log.Warn().Err(err).Str("file", e.V.Build.Name).Msg("Failed to delete old executable file")
                        }
                    }

                    // Rename temporary file to official file
                    if err := os.Rename(tempExecName, e.V.Build.Name); err != nil {
                        log.Err(err).Str("temp_file", tempExecName).Str("target_file", e.V.Build.Name).Msg("Failed to rename file")
                        // Clean up temporary file
                        os.Remove(tempExecName)
                        cmd = nil
                        return
                    }

                    log.Info().Str("file", e.V.Build.Name).Msg("Executable file updated successfully, starting new program")

                    if isFirstRun {
                        isFirstRun = false
                    }

                    Running(input)
                    cmd = nil
                }()
            })

            // Start countdown display
            startCountdown(e.V.Build.Delay)

            // Clear all pending change notifications
            consume(change)

        default:
            time.Sleep(time.Second)
        }
    }
}
func Watching(input e.Run) {
    // Clean temporary files on startup
    cleanupTempFiles()

    // Set up signal handling to clean temporary files on program exit
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        log.Info().Msg("Received exit signal, cleaning temporary files...")
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
                // log.Info().Str("event", event.Name).Send()
                change <- true
            case err2 := <-watcher.Errors:
                log.Err(err2).Msg("Monitoring failed")
            }
        }
    }()
    for s := range watchDirs {
        watcher.Add(s)
    }
    // 20241218 Trigger one first by Node
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
