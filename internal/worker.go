package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gammazero/workerpool"
)

const (
	// TODO: fetch from go release
	latest = "1.14"
	goMod  = "go.mod"
)

type controller interface {
	Submit() error
	Prepare() error
	Cleanup() error
}

type file struct {
	path string
	name string
}

type Worker struct {
	path        string
	currentGo   string
	files       []file
	vController controller
}

func NewWorker(url string) Worker {
	cCtr := NewWorkerVC(url)
	return Worker{
		path:        url,
		vController: &cCtr,
	}
}

func (w Worker) Init() {
	var wg sync.WaitGroup
	wp := workerpool.New(30)

	for i := 0; i < 1; i++ {
		wg.Add(1)
		wp.Submit(func() {
			w.bump(&wg)
		})
	}

	wg.Wait()
}

// TODO: check if hub installed
// TODO: compare values of old vs new go version
func (w Worker) bump(wg *sync.WaitGroup) {
	//if err := w.vController.Prepare(); err != nil {
	//	haltOnError(err)
	//}

	if err := filepath.Walk(w.path, w.visit); err != nil {
		haltOnError(err)
	}

	// no go.mod with version exit
	if w.currentGo == "" {
		return
	}

	if err := w.vendor(); err != nil {
		haltOnError(err)
	}

	if err := w.visitGitHub(); err != nil {
		haltOnError(err)
	}

	// TODO: check if hub installed
	if err := w.vController.Submit(); err != nil {
		fmt.Println("submit")
		haltOnError(err)
	}

	//if err := w.vController.Cleanup(); err != nil {
	//	fmt.Println("cleanup")
	//	haltOnError(err)
	//}

	wg.Done()
}

func (w *Worker) visit(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() && fi.Name() == "vendor" {
		return filepath.SkipDir
	}

	if fi.IsDir() && fi.Name() == ".git" {
		return filepath.SkipDir
	}

	w.files = append(w.files, file{path: path, name: fi.Name()})

	if fi.IsDir() {
		return nil
	}

	matched, err := filepath.Match("go.mod", fi.Name())
	if err != nil {
		return err
	}

	if matched {
		if err := w.editFile(path); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) visitGitHub() error {
	for _, file := range w.files {
		matched, err := filepath.Match("*.yaml", file.name)
		if err != nil {
			return err
		}

		if matched {
			if err := w.editFile(file.path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Worker) editFile(path string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if strings.Contains(path, goMod) {
		w.storeCurrentGoVersion(read)
	}

	replaced := bytes.Replace(read, []byte(w.currentGo), []byte(latest), -1)
	err = ioutil.WriteFile(path, replaced, 0)

	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) storeCurrentGoVersion(read []byte) {
	// Find `go 1.13` index to extract go version
	index := bytes.Index(read, []byte("go "))
	if index != -1 {
		index = index + 3
		// TODO: leave naive approach
		w.currentGo = bytes.NewBuffer(read[index : index+4]).String()
	}
}

func (w *Worker) vendor() error {
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func haltOnError(err error) {
	if err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
