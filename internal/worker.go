package internal

import (
	"bytes"
	"fmt"
	"github.com/gammazero/workerpool"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	// TODO: fetch from go release
	latest = "1.14"
)

type file struct {
	path string
	name string
	currentGo string
}

type Worker struct {
	path       string
	files      []file
}

func NewWorker(url string) Worker {
	return Worker{
		path:       url,
	}
}

func (w Worker) Init() {
	var wg sync.WaitGroup
	wp := workerpool.New(30)

	for i := 0; i < 1; i++ {
		wg.Add(1)
		wp.Submit(func (){
			w.bump(&wg)
		})
	}

	wg.Wait()
}

func (w Worker) bump(wg *sync.WaitGroup) {
	// TODO: compare values of old vs new go version
	if err := filepath.Walk(w.path, w.visit); err != nil {
		haltOnError(err)
	}

	if err := w.vendor(); err != nil {
		haltOnError(err)
	}

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
		fmt.Println(err)
		return err
	}

	if matched {
		if err := w.editFile(path); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) editFile(path string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// TODO: if go.mod get currentGo version and store for .github edits
	replaced := bytes.Replace(read, []byte("1.13"), []byte(latest), -1)
	err = ioutil.WriteFile(path, replaced, 0)

	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) vendor() error {
	cmd := exec.Command(
		"go",
		"mod",
		"vendor",
	)
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func haltOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
