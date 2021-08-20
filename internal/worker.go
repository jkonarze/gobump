package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	goMod      = "go.mod"
	makefile   = "Makefile"
	dockerfile = "Dockerfile"
	dockerfileProto = "Dockerfile-proto"
	dockerfileLinter = "Dockerfile-linter"
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
	path           string
	version        string
	currentGo      string
	builderVersion string
	buildImage     string
	runImage       string
	files          []file
	vController    controller
}

func NewWorker(path, version, builder, buildImage, runImage string) Worker {
	return Worker{
		path:           path,
		version:        version,
		builderVersion: builder,
		buildImage:     buildImage,
		runImage:       runImage,
	}
}

func (w *Worker) Init() {
	//var wg sync.WaitGroup
	//wp := workerpool.New(30)
	//repos, err := w.repos()
	//
	//if err != nil {
	//	w.haltOnError(err, "init")
	//}
	//
	//for _, v := range repos {
	//	path := filepath.Join(w.path, v)
	//	wg.Add(1)
	//	wp.Submit(func() {
	//		w.bump(&wg, path)
	//
	//	})
	//}
	//
	//wg.Wait()
	//wp.Stop()

	repos, err := w.repos()

	if err != nil {
		w.haltOnError(err, "init")
	}

	for _, v := range repos {
		path := filepath.Join(w.path, v)
		w.bump(path)
	}
}

func (w *Worker) repos() ([]string, error) {
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
	return repos, nil
}

// TODO: check if hub installed
// TODO: compare values of old vs new go version
func (w *Worker) bump(path string) {
	vCli := NewWorkerVC(path)
	w.vController = &vCli
	if err := w.vController.Prepare(); err != nil {
		w.haltOnError(err, "prepare")
	}

	fmt.Printf("🚧 working on %v\n", path)
	if err := filepath.Walk(path, w.visit); err != nil {
		w.haltOnError(err, "walk")
	}

	// no go.mod with version exit
	if w.currentGo == "" {
		return
	}

	if err := w.vendor(path); err != nil {
		w.haltOnError(err, "vendor")
	}

	if err := w.visitGitHub(); err != nil {
		w.haltOnError(err, "github")
	}

	// TODO: check if hub installed
	if err := w.vController.Submit(); err != nil {
		w.haltOnError(err, "submit")
	}

	if err := w.vController.Cleanup(); err != nil {
		w.haltOnError(err, "cleanup")
	}
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
		if err := w.editGoMod(path); err != nil {
			return err
		}
	}

	if strings.Contains(path, makefile) {
		err := w.updateBuilderVersion(path)

		if err != nil {
			return err
		}
	}

	if strings.Contains(path, dockerfile) &&
		!strings.Contains(path, dockerfileProto) &&
		!strings.Contains(path, dockerfileLinter) {
		err := w.updateGoBuildImage(path)

		if err != nil {
			return err
		}

		err = w.updateGoRunImage(path)

		if err != nil {
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
			if err := w.editGoMod(file.path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Worker) editGoMod(path string) error {
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
	s := "go "
	index := bytes.Index(read, []byte(s))
	if index != -1 {
		index = index + len(s)
		// TODO: leave naive approach
		versionTagSize := 4
		w.currentGo = bytes.NewBuffer(read[index : index+versionTagSize]).String()
	}
}

func (w *Worker) updateBuilderVersion(path string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	s := "BUILD_TOOLS_VERSION="
	index := bytes.Index(read, []byte(s))
	if index != -1 {
		index = index + len(s)
		versionTagSize := 6
		// TODO: leave naive approach
		currentBuilder := bytes.NewBuffer(read[index : index+versionTagSize]).String()

		replaced := bytes.Replace(read, []byte(currentBuilder), []byte(w.builderVersion), -1)
		err := ioutil.WriteFile(path, replaced, 0)

		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) updateGoBuildImage(path string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	s := "FROM golang:"
	buildIndex := bytes.Index(read, []byte(s))
	if buildIndex != -1 {
		buildIndex = buildIndex + len(s)
		versionTagSize := 4
		// TODO: leave naive approach
		currentBuild := bytes.NewBuffer(read[buildIndex : buildIndex+versionTagSize]).String()
		replaced := bytes.Replace(read, []byte(currentBuild), []byte(w.buildImage), -1)
		err := ioutil.WriteFile(path, replaced, 0)

		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) updateGoRunImage(path string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	s := "FROM alpine:"
	runIndex := bytes.Index(read, []byte(s))
	if runIndex != -1 {
		runIndex = runIndex + len(s)
		versionTagSize := 4
		// TODO: leave naive approach
		currentRun := bytes.NewBuffer(read[runIndex : runIndex+versionTagSize]).String()

		replaced := bytes.Replace(read, []byte(currentRun), []byte(w.runImage), -1)
		err := ioutil.WriteFile(path, replaced, 0)

		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) vendor(path string) error {
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = filepath.Join(path)


	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (w *Worker) haltOnError(err error, source string) {
	if err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			fmt.Printf("🚨 err %v\n", err)
		}
		fmt.Printf("🚨 err %v source %v path %v\n", err, source, w.path)
		os.Exit(1)
	}
}
