package testrunner_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/fwojciec/testrunner"
)

type mockRunner struct {
	calls []string
	mu    sync.RWMutex
}

func (r *mockRunner) RunTest(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, path)
	return nil
}

func (r *mockRunner) Calls() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls
}

var (
	testDebounceDelay = 10 * time.Millisecond
	testPollInterval  = 10 * time.Millisecond
)

func TestRunsOnlyOnGoFileWrites(t *testing.T) {
	t.Parallel()
	th := setup()
	defer th.TeardownFn()
	_ = createAndWriteToFile(filepath.Join(th.TestDir, "file.go"))
	_ = createAndWriteToFile(filepath.Join(th.TestDir, "file.txt"))
	time.Sleep(30 * time.Millisecond)
	equals(t, []string{th.TestDir}, th.Runner.Calls())
}

func TestDiscoversWritesInNewSubdirectories(t *testing.T) {
	t.Parallel()
	th := setup()
	defer th.TeardownFn()
	subDir := "another"
	_ = createAndWriteToFile(filepath.Join(th.TestDir, "file.go"))
	_ = os.Mkdir(filepath.Join(th.TestDir, subDir), 0777)
	time.Sleep(20 * time.Millisecond)
	_ = createAndWriteToFile(filepath.Join(th.TestDir, subDir, "another.go"))
	time.Sleep(30 * time.Millisecond)
	equals(t, []string{th.TestDir, filepath.Join(th.TestDir, subDir)}, th.Runner.Calls())
}

type testHelper struct {
	TestDir    string
	TeardownFn func() error
	Runner     *mockRunner
	Debouncer  *testrunner.Debouncer
	Actor      *testrunner.Actor
}

func setup() *testHelper {
	testDir, err := ioutil.TempDir("", "testrunner")
	if err != nil {
		panic(err)
	}
	tr := &mockRunner{}
	db := testrunner.NewDebouncer(tr, testDebounceDelay)
	go db.Run()
	a := testrunner.NewActor(testDir, testPollInterval)
	go a.Run(db.Pathc)
	time.Sleep(10 * time.Millisecond)
	return &testHelper{
		TestDir: testDir,
		TeardownFn: func() error {
			a.Stop()
			db.Stop()
			return os.RemoveAll(testDir)
		},
		Runner:    tr,
		Debouncer: db,
		Actor:     a,
	}
}

func createAndWriteToFile(fname string) error {
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return err
	}
	if _, err := f.WriteString("test"); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
