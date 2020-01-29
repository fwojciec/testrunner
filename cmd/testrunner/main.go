package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fwojciec/testrunner"
	"github.com/oklog/run"
)

func main() {
	var g run.Group
	{
		wa := testrunner.New(".")
		g.Add(func() error {
			return wa.Run()
		}, func(error) {
			wa.Stop()
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
		return errors.New("canceled")
	}
}
