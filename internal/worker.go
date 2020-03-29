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
	goMod = "go.mod"
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
	version     string
	currentGo   string
	files       []file
	vController controller
}

func NewWorker(path, version string) Worker {
	return Worker{
		path:        path,
		version:     version,
	}
}

func (w Worker) Init() {
	var wg sync.WaitGroup
	wp := workerpool.New(30)
	repos, err := w.repos()

	if err != nil {
		haltOnError(err)
	}

	for _, v := range repos{
		path := filepath.Join(w.path, v)
		fmt.Println(path)
		wg.Add(1)
		wp.Submit(func() {
			w.bump(&wg, path)

		})
	}

	wg.Wait()
	wp.Stop()
}

func (w Worker) repos() ([]string, error) {
	var repos []string
	files, err := ioutil.ReadDir(w.path)
	if err != nil {
		return repos, err
	}
	for _, file := range files {
		if file.IsDir() {
			repos = append(repos, file.Name())
		}
	}
	fmt.Println("counting files ", len(repos))
	return repos, nil
}

// TODO: check if hub installed
// TODO: compare values of old vs new go version
func (w Worker) bump(wg *sync.WaitGroup, path string) {
	vCli := NewWorkerVC(path)
	w.vController = &vCli
	//if err := w.vController.Prepare(); err != nil {
	//	haltOnError(err)
	//}

	if err := filepath.Walk(path, w.visit); err != nil {
		haltOnError(err)
	}

	// no go.mod with version exit
	if w.currentGo == "" {
		return
	}

	if err := w.vendor(path); err != nil {
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

	replaced := bytes.Replace(read, []byte(w.currentGo), []byte(w.version), -1)
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

func (w *Worker) vendor(path string) error {
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = filepath.Join(path)

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
