package testrunner_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/fwojciec/testrunner"
)

func TestDebouncer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		paths []string
		exp   []string
	}{
		{[]string{"a", "a", "a", "b"}, []string{"a", "b"}},
		{[]string{"a", "a", "b", "a"}, []string{"a", "b"}},
		{[]string{"a", "b", "a", "b"}, []string{"a", "b"}},
		{[]string{"a", "b", "c", "b"}, []string{"a", "b", "c"}},
	}
	for i, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			runner := &mockRunner{}
			db := testrunner.NewDebouncer(runner, 2*time.Millisecond)
			go db.Run()
			defer db.Stop()
			for _, path := range tc.paths {
				db.Pathc <- path
			}
			time.Sleep(10 * time.Millisecond)
			equals(t, tc.exp, runner.Calls())
		})
	}
}
