package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type task struct {
	id         string
	title      string
	status     bool
	priority   int
	tags       []string
	discretion string
}

func (t *task) generateID() {
	// Nanosecond timestamp hex-encoded (~16 chars)
	ts := time.Now().UnixNano()

	// Crypto-random string prefix/suffix
	b := make([]byte, 4)
	rand.Read(b)
	t.id = fmt.Sprintf("%x-%x", ts, b)
}

func (t task) save(dirPath string) error {
	taskDir := path.Join(dirPath, t.id)

	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}

	filePath := path.Join(taskDir, "TASK.md")

	fd, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fd.Close()

	status := "OPEN"
	if !t.status {
		status = "CLOSE"
	}
	tags := strings.Join(t.tags, " ,")

	_, err = fmt.Fprintf(fd, "# %s\n\n- STATUS: %s\n- PRIORITY: %d\n- TAGS: %s\n\n%s", t.title, status, t.priority, tags, t.discretion)
	return err
}

func main() {
}

func getRootDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ".task", nil
	}

	root := strings.TrimSpace(string(out))
	dir := path.Join(root, ".tasks")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}
