package testrunner

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchActor performs actions.
type WatchActor struct {
	rootDir          string
	watcher          *fsnotify.Watcher
	quitc            chan chan struct{}
	refreshDirsDelay time.Duration
	stderr           io.Writer
	stdout           io.Writer
}

// Run runs the testrunner.
func (a *WatchActor) Run() error {
	for {
		select {
		case event := <-a.watcher.Events:
			if event.Op == fsnotify.Write {
				testArg := "./" + filepath.Dir(event.Name)
				cmd := exec.Command("go", "test", "-race", testArg)
				cmd.Stdout = a.stdout
				cmd.Stderr = a.stderr
				if err := cmd.Run(); err != nil {
					fmt.Fprintln(a.stderr, err)
				}
			}
		case err := <-a.watcher.Errors:
			fmt.Fprintln(a.stderr, err)
		case <-time.After(a.refreshDirsDelay):
			// fsnotify automatically removes deleted dirs so we don't have to
			// manually remove anything. we can just re-add everything since
			// internally fsnotify just stores watched dirs as a map.
			dirs, err := getGoDirs(a.rootDir)
			if err != nil {
				fmt.Fprintln(a.stderr, err)
			}
			for _, dir := range dirs {
				if err := a.watcher.Add(dir); err != nil {
					fmt.Fprintln(a.stderr, err)
				}
			}
		case c := <-a.quitc:
			a.watcher.Close()
			close(c)
			return nil
		}
	}
}

// Stop stopes the WatchActor.
func (a *WatchActor) Stop() {
	c := make(chan struct{})
	a.quitc <- c
	<-c
}

// New returns a new instance of WatchActor.
func New(rootDir string) *WatchActor {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	wa := &WatchActor{
		rootDir:          rootDir,
		refreshDirsDelay: 10 * time.Second,
		watcher:          watcher,
		quitc:            make(chan chan struct{}),
		stderr:           os.Stderr,
		stdout:           os.Stdout,
	}
	dirs, err := getGoDirs(rootDir)
	if err != nil {
		panic(err)
	}
	for _, dir := range dirs {
		if err := wa.watcher.Add(dir); err != nil {
			panic(err)
		}
	}
	return wa
}

func getGoDirs(rootDir string) ([]string, error) {
	var res []string
	if err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsDir() {
			files, err := ioutil.ReadDir(path)
			if err != nil {
				return err
			}
			for _, f := range files {
				if filepath.Ext(f.Name()) == ".go" {
					res = append(res, filepath.Clean(path))
					break
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}
