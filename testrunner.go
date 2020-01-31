package testrunner

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Runner runs the test command.
type Runner interface {
	RunTest(path string) error
}

// TestRunner implements the Runner interface
type TestRunner struct {
	Stderr io.Writer
	Stdout io.Writer
}

// RunTest runs the test command.
func (t *TestRunner) RunTest(path string) error {
	cmd := exec.Command("go", "test", "-race", path)
	cmd.Stdout = t.Stdout
	cmd.Stderr = t.Stderr
	return cmd.Run()
}

// Actor performs actions.
type Actor struct {
	PollInterval time.Duration // polling for directory structure changes
	RootDir      string        // root directory of the project

	watcher *fsnotify.Watcher
	quitc   chan chan struct{}
	stderr  io.Writer
	stdout  io.Writer
}

// NewActor returns a new instance of Actor.
func NewActor(rootDir string, pollInterval time.Duration) *Actor {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	wa := &Actor{
		RootDir:      rootDir,
		PollInterval: pollInterval,
		watcher:      watcher,
		quitc:        make(chan chan struct{}),
		stderr:       os.Stderr,
		stdout:       os.Stdout,
	}
	if err := wa.addGoDirs(); err != nil {
		panic(err)
	}
	return wa
}

// Run runs the testrunner.
func (a *Actor) Run(pathc chan<- string) error {
	t := time.NewTicker(a.PollInterval)
	for {
		select {
		case event := <-a.watcher.Events:
			if event.Op == fsnotify.Write && isGoFile(event.Name) {
				pathc <- normalizePath(filepath.Dir(event.Name))
			}
		case err := <-a.watcher.Errors:
			fmt.Fprintln(a.stderr, err)
		case <-t.C:
			if err := a.addGoDirs(); err != nil {
				fmt.Fprintln(a.stderr, err)
			}
		case c := <-a.quitc:
			defer close(c)
			t.Stop()
			return a.watcher.Close()
		}
	}
}

func (a *Actor) addGoDirs() error {
	return filepath.Walk(a.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsDir() && dirContainsGoFiles(path) {
			if err := a.watcher.Add(path); err != nil {
				return err
			}
		}
		return nil
	})
}

// Stop stopes the WatchActor.
func (a *Actor) Stop() {
	c := make(chan struct{})
	a.quitc <- c
	<-c
}

func isGoFile(fname string) bool {
	return strings.ToLower(filepath.Ext(fname)) == ".go"
}

func dirContainsGoFiles(path string) bool {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return false
	}
	for _, f := range files {
		if isGoFile(f.Name()) {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	if filepath.IsAbs(path) || strings.HasPrefix(path, ".") {
		return path
	}
	return "./" + path
}
