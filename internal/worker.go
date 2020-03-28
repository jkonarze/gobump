package internal

import (
	"bytes"
	"fmt"
	"github.com/gammazero/workerpool"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// TODO: fetch from go release
	latest = "1.14"
	goMod = "go.mod"
)

type file struct {
	path string
	name string
}

type Worker struct {
	path       string
	currentGo string
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

	if err := filepath.Walk(w.path, w.visitGitHub); err != nil {
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

// TODO: optimize so only visits the .github folder
func (w *Worker) visitGitHub(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() && fi.Name() != "vendor" {
		return filepath.SkipDir
	}

	if fi.IsDir() && fi.Name() == ".git" {
		return filepath.SkipDir
	}

	if fi.IsDir() {
		return nil
	}

	fmt.Println(fi.Name())
	matched, err := filepath.Match("*", fi.Name())
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
