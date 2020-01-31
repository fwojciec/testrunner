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
	testDebounceDelay = 1 * time.Millisecond
	testPollInterval  = 1 * time.Millisecond
	wait              = 100 * time.Millisecond
)

func TestRunsOnGoFileWrites(t *testing.T) {
	t.Parallel()
	th := setup()
	defer th.TeardownFn()
	_ = createAndWriteToFile("file.go", th.TestDir)
	time.Sleep(wait)
	equals(t, []string{th.TestDir}, th.Runner.Calls())
}

func TestSkipsNonGoFileWrites(t *testing.T) {
	t.Parallel()
	th := setup()
	defer th.TeardownFn()
	_ = createAndWriteToFile("file.tst", th.TestDir)
	time.Sleep(wait)
	equals(t, 0, len(th.Runner.Calls()))
}

func TestDiscoversWritesInNewSubdirectories(t *testing.T) {
	t.Parallel()
	th := setup()
	defer th.TeardownFn()
	_ = createAndWriteToFile("file.go", th.TestDir)
	subDir := filepath.Join(th.TestDir, "another")
	_ = os.Mkdir(subDir, 0777)
	f, _ := os.Create(filepath.Join(subDir, "another.go"))
	defer f.Close()
	_ = f.Sync()
	time.Sleep(wait)
	f.Write([]byte("more stuff"))
	f.Sync()
	time.Sleep(wait)
	equals(t, []string{th.TestDir, subDir}, th.Runner.Calls())
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
	time.Sleep(wait)
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

func createAndWriteToFile(fname, dir string) error {
	f, err := os.OpenFile(filepath.Join(dir, fname), os.O_WRONLY|os.O_CREATE, 0666)
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
