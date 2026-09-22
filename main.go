package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

type task struct {
	id          string
	title       string
	done        bool
	priority    int
	tags        []string
	description string
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
	if t.id == "" {
		t.generateID()
	}
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
	if t.done {
		status = "CLOSE"
	}
	tags := strings.Join(t.tags, " ,")

	_, err = fmt.Fprintf(fd, "# %s\n\n- STATUS: %s\n- PRIORITY: %d\n- TAGS: %s\n\n%s", t.title, status, t.priority, tags, t.description)
	return err
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "new" {
			t, err := newTask()
			if err != nil {
				panic(err)
			}
			rootPath, _ := getRootDir()
			t.save(rootPath)
		}
	}
	listTask()
}

func newTask() (*task, error) {
	t := task{}
	t.generateID()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("title: ")
	title, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	t.title = strings.TrimSpace(title)

	fmt.Print("description: ")
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lines = append(lines, line)
	}
	t.description = strings.Join(lines, "\n")

	fmt.Print("priority (default: 50): ")
	priorityStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	priorityStr = strings.TrimSpace(priorityStr)
	if priorityStr == "" {
		t.priority = 50
	} else if p, err := strconv.Atoi(priorityStr); err == nil {
		t.priority = p
	}

	return &t, nil
}

func listTask() {
	rootPath, err := getRootDir()
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := loadTask(rootPath, entry.Name())
		if err != nil {
			continue
		}

		status := "OPEN"
		if !t.done {
			status = "CLOSE"
		}
		tags := strings.Join(t.tags, ",")

		fmt.Printf("%s\t[%s]\t%d\t%s\t%s\n", t.id, status, t.priority, t.title, tags)
	}
}

func loadTask(rootPath, id string) (*task, error) {
	filePath := path.Join(rootPath, id, "TASK.md")
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	t := &task{id: id}
	scanner := bufio.NewScanner(fd)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "# "):
			t.title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "- STATUS:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "- STATUS:"))
			t.done = v == "OPEN"
		case strings.HasPrefix(line, "- PRIORITY:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "- PRIORITY:"))
			if p, err := strconv.Atoi(v); err == nil {
				t.priority = p
			}
		case strings.HasPrefix(line, "- TAGS:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "- TAGS:"))
			if v != "" {
				parts := strings.Split(v, ",")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				t.tags = parts
			}
		}
		// description lines are intentionally not read further; we stop
		// caring about content once metadata is parsed since listTask
		// doesn't need it.
	}

	return t, scanner.Err()
}

func getRootDir() (string, error) {
	var root string
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		root = ".task"
	} else {
		root = path.Join(strings.TrimSpace(string(out)), ".tasks")
	}

	err = os.MkdirAll(root, 0755)
	return root, err
}
