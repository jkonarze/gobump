package internal

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

const (
	bumpBranch = "nextGo"
	master     = "master"
)

type WorkerVC struct {
	path      string
	originalB string
}

func NewWorkerVC(path string) WorkerVC {
	return WorkerVC{
		path: path,
	}
}

func (w *WorkerVC) Prepare() error {
	if err := w.currentBranch(); err != nil {
		return err
	}

	if err := w.stash(); err != nil {
		return err
	}

	if err := w.checkout(master); err != nil {
		return err
	}

	if err := w.pull(); err != nil {
		return err
	}

	return nil
}

func (w *WorkerVC) currentBranch() error {
	cmd := exec.Command("hub", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = filepath.Join(w.path)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	w.originalB = string(output)
	return nil
}

func (w *WorkerVC) stash() error {
	cmd := exec.Command("hub", "stash")
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) checkout(branch string) error {
	cmd := exec.Command("hub", "checkout", branch)
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) pull() error {
	cmd := exec.Command("hub", "pull")
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) Submit() error {
	if err := w.addChanges(); err != nil {
		fmt.Println("add", err)
		return err
	}

	if err := w.branchOut(); err != nil {
		fmt.Println("branch out ", err)
		return err
	}

	if err := w.commit(); err != nil {
		fmt.Println("commit ", err)
		return err
	}

	if err := w.pr(); err != nil {
		fmt.Println("pr ", err)
		return err
	}

	return nil
}

func (w *WorkerVC) branchOut() error {
	// TODO: issue when passing constant as a branch?
	cmd := exec.Command("hub", "checkout", "-b", "next-Go")
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) commit() error {
	cmd := exec.Command("hub", "commit", `-am "bump go version with go-bump"`)
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) pr() error {
	cmd := exec.Command("hub",
		"pull-request",
		"-p",
		"-l",
		"minor",
		"--no-edit",
		"-m",
		"update go version",
	)
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) addChanges() error {
	cmd := exec.Command("hub", "add", ".")
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) Cleanup() error {
	if err := w.checkout(w.originalB); err != nil {
		fmt.Println("checkout")
		return err
	}

	if err := w.stashPop(); err != nil {
		fmt.Println("stash pop")
		return err
	}

	if err := w.removeBranch(); err != nil {
		fmt.Println("remove branch")
		return err
	}

	return nil
}

func (w *WorkerVC) stashPop() error {
	cmd := exec.Command("hub", "stash", "pop")
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (w *WorkerVC) removeBranch() error {
	cmd := exec.Command("hub", "branch", "-D", bumpBranch)
	cmd.Dir = filepath.Join(w.path)

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
