package testrunner

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Debouncer runs a command once, after a delay. It maintains a simple queue of
// unique requests so that debounce logic can distinguish between different
// types of requests.
type Debouncer struct {
	Pathc   chan string
	runner  Runner
	waiting map[string]bool
	queue   []string
	quitc   chan chan struct{}
	d       time.Duration
	stdErr  io.Writer
}

// NewDebouncer returns a new instance of DebouncedRunned.
func NewDebouncer(r Runner, d time.Duration) *Debouncer {
	db := &Debouncer{
		Pathc:   make(chan string),
		runner:  r,
		d:       d,
		queue:   make([]string, 0),
		quitc:   make(chan chan struct{}),
		waiting: make(map[string]bool, 0),
		stdErr:  os.Stderr,
	}
	return db
}

// Run runs the debouncer.
func (d *Debouncer) Run() error {
	t := time.NewTicker(d.d)
	t.Stop()
	for {
		select {
		case path := <-d.Pathc:
			if _, ok := d.waiting[path]; !ok {
				d.push(path)
			}
			t.Stop()
			t = time.NewTicker(d.d)
		case <-t.C:
			if len(d.queue) < 1 {
				t.Stop()
				continue
			}
			path := d.pop()
			if err := d.runner.RunTest(path); err != nil {
				fmt.Fprintln(d.stdErr, err)
			}
		case c := <-d.quitc:
			t.Stop()
			close(c)
			return nil
		}
	}
}

func (d *Debouncer) push(path string) {
	d.queue = append(d.queue, path)
	d.waiting[path] = true
}

func (d *Debouncer) pop() string {
	var path string
	path, d.queue = d.queue[0], d.queue[1:]
	delete(d.waiting, path)
	return path
}

// Stop stops the runned.
func (d *Debouncer) Stop() {
	c := make(chan struct{})
	d.quitc <- c
	<-c
}
