package watch

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/qwenode/sohot/boot"
	"github.com/qwenode/sohot/i18n"
	"github.com/qwenode/sohot/types"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// Reloader manages the hot-reload lifecycle
type Reloader struct {
	config     types.Run
	change     chan struct{}
	stop       chan struct{}
	mu         sync.Mutex
	process    *exec.Cmd
	isFirstRun bool
}

// New creates a new Reloader instance
func New(config types.Run) *Reloader {
	return &Reloader{
		config:     config,
		change:     make(chan struct{}, 100),
		stop:       make(chan struct{}),
		isFirstRun: true,
	}
}

// Start begins the hot-reload process
func (r *Reloader) Start() {
	cleanupTempFiles()
	r.setupSignalHandler()
	r.startWatcher()

	// Trigger initial build/run
	r.change <- struct{}{}

	if r.config.Only {
		r.runOnlyLoop()
	} else {
		r.buildLoop()
	}
}

// setupSignalHandler sets up cleanup on exit
func (r *Reloader) setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info().Msg(i18n.T("watch.shutdown_signal"))
		r.stopProcess()
		cleanupTempFiles()
		os.Exit(0)
	}()
}

// startWatcher starts the file system watcher
func (r *Reloader) startWatcher() {
	dirs := r.collectWatchDirs()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg(i18n.T("watch.watcher_init_failed"))
	}

	for dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			log.Warn().Err(err).Str("directory", dir).Msg(i18n.T("watch.watch_dir_failed"))
		}
	}

	go r.watchEvents(watcher)
}

// collectWatchDirs collects directories to watch
func (r *Reloader) collectWatchDirs() map[string]bool {
	dirs := make(map[string]bool)

	if r.config.Only {
		dirs[filepath.Dir(boot.V.Build.Name)] = true
		return dirs
	}

	for _, include := range boot.V.Watch.Include {
		filepath.WalkDir(include, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg(i18n.T("watch.dir_traversal_error"))
				return nil
			}
			if d != nil && d.IsDir() && !isExcluded(path) {
				dirs[path] = true
			}
			return nil
		})
	}

	return dirs
}

// watchEvents handles file system events
func (r *Reloader) watchEvents(watcher *fsnotify.Watcher) {
	for {
		select {
		case event := <-watcher.Events:
			if r.shouldIgnoreEvent(event.Name) {
				continue
			}
			r.notifyChange()
		case err := <-watcher.Errors:
			log.Error().Err(err).Msg(i18n.T("watch.watcher_error"))
		}
	}
}

// shouldIgnoreEvent checks if an event should be ignored
func (r *Reloader) shouldIgnoreEvent(name string) bool {
	if !r.config.Only && isExcluded(name) {
		return true
	}
	stat, err := os.Stat(name)
	if err != nil || stat.IsDir() {
		return true
	}
	return false
}

// notifyChange sends a change notification (non-blocking)
func (r *Reloader) notifyChange() {
	select {
	case r.change <- struct{}{}:
	default:
	}
}

// runOnlyLoop handles the "only" mode (restart without rebuild)
func (r *Reloader) runOnlyLoop() {
	for {
		select {
		case <-r.change:
			r.drainChanges()
			time.Sleep(time.Second)

			if !fileExists(boot.V.Build.Name) {
				log.Warn().Str("executable", boot.V.Build.Name).Msg(i18n.T("watch.executable_not_found"))
				continue
			}

			log.Info().Msg(i18n.T("watch.change_detected_restart"))
			r.restart()

		case <-r.stop:
			return
		}
	}
}

// buildLoop handles the build-and-run mode
func (r *Reloader) buildLoop() {
	var debounceTimer *time.Timer

	for {
		select {
		case <-r.change:
			log.Info().Msg(i18n.T("watch.source_change_detected"))

			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			delay := time.Duration(boot.V.Build.Delay) * time.Millisecond
			debounceTimer = time.AfterFunc(delay, func() {
				r.drainChanges()
				if err := r.build(); err != nil {
					log.Error().Err(err).Msg(i18n.T("watch.build_failed"))
					return
				}
				r.restart()
			})

		case <-r.stop:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

// build compiles the application
func (r *Reloader) build() error {
	log.Info().Msg(i18n.T("watch.build_started"))

	tempFile := boot.V.Build.Name + ".tmp"
	cleanupTempBuildFiles()

	args := []string{"build"}
	args = append(args, boot.V.Build.Command...)
	args = append(args, "-o", tempFile, boot.V.Build.Package)

	log.Debug().Strs("args", args).Msg(i18n.T("watch.build_executing"))

	cmd := exec.Command("go", args...)
	cmd.Stdout = boot.StdoutWriter
	cmd.Stderr = boot.StderrWriter

	if err := cmd.Run(); err != nil {
		os.Remove(tempFile)
		return err
	}

	if !fileExists(tempFile) {
		return nil
	}

	log.Info().Msg(i18n.T("watch.build_succeeded"))

	// Stop old process before replacing executable
	r.stopProcess()
	time.Sleep(100 * time.Millisecond)

	// Replace executable
	if fileExists(boot.V.Build.Name) {
		forceDeleteFile(boot.V.Build.Name)
	}

	if err := os.Rename(tempFile, boot.V.Build.Name); err != nil {
		log.Error().Err(err).Str("target", boot.V.Build.Name).Msg(i18n.T("watch.executable_update_failed"))
		os.Remove(tempFile)
		return err
	}

	return nil
}

// restart stops the old process and starts a new one
func (r *Reloader) restart() {
	r.mu.Lock()
	first := r.isFirstRun
	r.isFirstRun = false
	r.mu.Unlock()

	if !first {
		r.stopProcess()
		time.Sleep(100 * time.Millisecond)
	}

	r.run()
}

// run starts the application process
func (r *Reloader) run() {
	cmd := exec.Command(boot.V.Build.Name, r.config.Command...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg(i18n.T("watch.stdout_capture_failed"))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error().Err(err).Msg(i18n.T("watch.stderr_capture_failed"))
		return
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg(i18n.T("watch.process_launch_failed"))
		return
	}

	r.mu.Lock()
	r.process = cmd
	r.mu.Unlock()

	go io.Copy(boot.StdoutWriter, stdout)
	go io.Copy(boot.StderrWriter, stderr)

	log.Info().Int("pid", cmd.Process.Pid).Msg(i18n.T("watch.process_started"))
}

// stopProcess terminates the running process
func (r *Reloader) stopProcess() {
	r.mu.Lock()
	proc := r.process
	r.process = nil
	r.mu.Unlock()

	if proc == nil || proc.Process == nil {
		return
	}

	log.Info().Int("pid", proc.Process.Pid).Msg(i18n.T("watch.process_terminating"))
	proc.Process.Kill()
	proc.Wait()
}

// drainChanges clears pending change notifications
func (r *Reloader) drainChanges() {
	for {
		select {
		case <-r.change:
		default:
			return
		}
	}
}

// isExcluded checks if a path should be excluded from watching
func isExcluded(path string) bool {
	path = strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	for _, exclude := range boot.V.Watch.Exclude {
		if strings.Contains(path, exclude) {
			return true
		}
	}
	return false
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// cleanupTempFiles removes temporary files from previous runs
func cleanupTempFiles() {
	patterns := []string{"*.delete_me_*", "*.tmp"}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, f := range matches {
			if err := os.Remove(f); err == nil {
				log.Debug().Str("file", f).Msg(i18n.T("watch.temp_file_removed"))
			}
		}
	}
}

// cleanupTempBuildFiles removes temporary build files
func cleanupTempBuildFiles() {
	matches, _ := filepath.Glob(boot.V.Build.Name + ".tmp*")
	for _, f := range matches {
		os.Remove(f)
	}
}

// forceDeleteFile attempts to delete a file, using rename strategy if needed
func forceDeleteFile(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}

	// Try rename strategy for locked files (Windows)
	tempName := path + ".delete_me_" + time.Now().Format("20060102150405")
	if err := os.Rename(path, tempName); err == nil {
		return nil
	}

	return os.Remove(path)
}

// Legacy API for backward compatibility
// Deprecated: Use New() and Start() instead

func Watching(input types.Run) {
	// No-op, handled by Reloader.Start()
}

func Building(input types.Run) {
	r := New(input)
	r.Start()
}
