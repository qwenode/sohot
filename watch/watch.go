package watch

import (
	"fmt"
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

type Reloader struct {
	config     types.Run
	change     chan struct{}
	mu         sync.Mutex
	process    *exec.Cmd
	done       chan struct{}
	isFirstRun bool
}

func New(config types.Run) *Reloader {
	return &Reloader{
		config:     config,
		change:     make(chan struct{}, 100),
		isFirstRun: true,
	}
}

func (r *Reloader) Start() {
	cleanupTempFiles()
	r.setupSignalHandler()
	r.startWatcher()

	r.change <- struct{}{}

	if r.config.Only {
		r.runOnlyLoop()
	} else {
		r.buildLoop()
	}
}

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

func (r *Reloader) startWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg(i18n.T("watch.watcher_init_failed"))
	}

	if r.config.Only {
		dir := filepath.Dir(boot.V.Build.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Warn().Err(err).Str("directory", dir).Msg(i18n.T("watch.watch_dir_failed"))
		}
		if err := watcher.Add(dir); err != nil {
			log.Warn().Err(err).Str("directory", dir).Msg(i18n.T("watch.watch_dir_failed"))
		}
	} else {
		for _, include := range boot.V.Watch.Include {
			addWatchTree(watcher, include)
		}
	}

	go r.watchEvents(watcher)
}

func addWatchTree(watcher *fsnotify.Watcher, root string) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg(i18n.T("watch.dir_traversal_error"))
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if isExcluded(path) {
			return fs.SkipDir
		}
		if err := watcher.Add(path); err != nil {
			log.Warn().Err(err).Str("directory", path).Msg(i18n.T("watch.watch_dir_failed"))
		}
		return nil
	})
}

func (r *Reloader) watchEvents(watcher *fsnotify.Watcher) {
	for {
		select {
		case event := <-watcher.Events:
			r.handleEvent(watcher, event)
		case err := <-watcher.Errors:
			log.Error().Err(err).Msg(i18n.T("watch.watcher_error"))
		}
	}
}

func (r *Reloader) handleEvent(watcher *fsnotify.Watcher, event fsnotify.Event) {
	if r.config.Only {
		if isBuildTarget(event.Name) {
			r.notifyChange()
		}
		return
	}

	if isBuildOutput(event.Name) || isExcluded(event.Name) {
		return
	}
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		r.notifyChange()
		return
	}

	stat, err := os.Stat(event.Name)
	if err != nil {
		return
	}
	if stat.IsDir() {
		if !event.Has(fsnotify.Create) {
			return
		}
		// WHY: files written before the new directory is watched produce no events; notifying here rebuilds with them.
		addWatchTree(watcher, event.Name)
	}
	r.notifyChange()
}

func (r *Reloader) notifyChange() {
	select {
	case r.change <- struct{}{}:
	default:
	}
}

func (r *Reloader) runOnlyLoop() {
	for range r.change {
		time.Sleep(time.Second)
		r.drainChanges()

		if !fileExists(boot.V.Build.Name) {
			log.Warn().Str("executable", boot.V.Build.Name).Msg(i18n.T("watch.executable_not_found"))
			continue
		}

		log.Info().Msg(i18n.T("watch.change_detected_restart"))
		r.restart()
	}
}

func (r *Reloader) buildLoop() {
	var debounceTimer *time.Timer
	var debounceC <-chan time.Time

	for {
		select {
		case <-r.change:
			delay := time.Duration(boot.V.Build.Delay) * time.Millisecond
			if debounceTimer == nil {
				log.Info().Dur("delay", delay).Msg(i18n.T("watch.source_change_detected"))
				debounceTimer = time.NewTimer(delay)
				debounceC = debounceTimer.C
				continue
			}

			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer.Reset(delay)

		case <-debounceC:
			r.drainChanges()
			if err := r.build(); err != nil {
				log.Error().Err(err).Msg(i18n.T("watch.build_failed"))
				debounceTimer = nil
				debounceC = nil
				continue
			}
			r.restart()
			debounceTimer = nil
			debounceC = nil
		}
	}
}

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
		return fmt.Errorf("go build finished without creating %s; check that build.package is a main package", tempFile)
	}

	log.Info().Msg(i18n.T("watch.build_succeeded"))

	r.stopProcess()
	time.Sleep(100 * time.Millisecond)

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

func (r *Reloader) restart() {
	r.mu.Lock()
	first := r.isFirstRun
	r.isFirstRun = false
	r.mu.Unlock()

	if !first {
		r.stopProcess()
		time.Sleep(100 * time.Millisecond)
		cleanupTempFiles()
	}

	r.run()
}

func (r *Reloader) run() {
	cmd := exec.Command(boot.V.Build.Name, r.config.Command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg(i18n.T("watch.process_launch_failed"))
		return
	}

	done := make(chan struct{})
	r.mu.Lock()
	r.process = cmd
	r.done = done
	r.mu.Unlock()

	log.Info().Int("pid", cmd.Process.Pid).Msg(i18n.T("watch.process_started"))

	go r.waitProcess(cmd, done)
}

func (r *Reloader) waitProcess(cmd *exec.Cmd, done chan struct{}) {
	err := cmd.Wait()

	r.mu.Lock()
	own := r.process == cmd
	if own {
		r.process = nil
		r.done = nil
	}
	r.mu.Unlock()

	if own {
		if err != nil {
			log.Warn().Int("pid", cmd.Process.Pid).Err(err).Msg(i18n.T("watch.process_exited"))
		} else {
			log.Info().Int("pid", cmd.Process.Pid).Msg(i18n.T("watch.process_exited"))
		}
	}
	close(done)
}

func (r *Reloader) stopProcess() {
	r.mu.Lock()
	proc, done := r.process, r.done
	r.process = nil
	r.done = nil
	r.mu.Unlock()

	if proc == nil || proc.Process == nil {
		return
	}

	log.Info().Int("pid", proc.Process.Pid).Msg(i18n.T("watch.process_terminating"))
	proc.Process.Kill()
	<-done
}

func (r *Reloader) drainChanges() {
	for {
		select {
		case <-r.change:
		default:
			return
		}
	}
}

func isExcluded(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	for _, segment := range strings.Split(p, "/") {
		if segment == ".git" || segment == ".idea" {
			return true
		}
	}
	if strings.HasSuffix(p, ".exe") {
		return true
	}
	s := p + "/"
	for _, exclude := range boot.V.Watch.Exclude {
		if exclude != "" && strings.Contains(s, exclude) {
			return true
		}
	}
	return false
}

func isBuildOutput(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	name := boot.V.Build.Name
	if name == "" {
		return false
	}
	return strings.EqualFold(abs, name) ||
		strings.EqualFold(abs, name+".tmp") ||
		strings.HasPrefix(strings.ToLower(abs), strings.ToLower(name+".delete_me_"))
}

func isBuildTarget(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil || boot.V.Build.Name == "" {
		return false
	}
	return strings.EqualFold(abs, boot.V.Build.Name)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cleanupTempFiles() {
	if boot.V.Build.Name == "" {
		return
	}

	files, _ := filepath.Glob(boot.V.Build.Name + ".delete_me_*")
	files = append(files, boot.V.Build.Name+".tmp")
	for _, f := range files {
		if err := os.Remove(f); err == nil {
			log.Debug().Str("file", f).Msg(i18n.T("watch.temp_file_removed"))
		}
	}
}

func cleanupTempBuildFiles() {
	os.Remove(boot.V.Build.Name + ".tmp")
}

func forceDeleteFile(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}

	// WHY: Windows refuses to delete an executable still held by a just-killed process but allows renaming it; without the rename, replacing the executable fails.
	tempName := path + ".delete_me_" + time.Now().Format("20060102150405")
	if err := os.Rename(path, tempName); err == nil {
		return nil
	}

	return os.Remove(path)
}
