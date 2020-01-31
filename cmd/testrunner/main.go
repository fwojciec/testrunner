package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fwojciec/testrunner"
	"github.com/oklog/run"
)

var (
	defaultPollInterval  = 10 * time.Second
	defaultDebounceDelay = 500 * time.Millisecond
	defaultRootDir       = "."
)

func main() {
	tr := &testrunner.TestRunner{
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	db := testrunner.NewDebouncer(tr, defaultDebounceDelay)
	a := testrunner.NewActor(defaultRootDir, defaultPollInterval)

	var g run.Group
	{
		g.Add(func() error {
			return db.Run()
		}, func(e error) {
			db.Stop()
		})
	}
	{
		g.Add(func() error {
			return a.Run(db.Pathc)
		}, func(error) {
			a.Stop()
		})
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(cancel)
		}, func(error) {
			close(cancel)
		})
	}
	fmt.Fprintln(os.Stderr, g.Run())
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-c:
		return fmt.Errorf("received signal %s", sig)
	case <-cancel:
		return fmt.Errorf("canceled")
	}
}
