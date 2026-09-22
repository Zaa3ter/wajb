package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "n", "new":
			t, err := newTask()
			if err != nil {
				panic(err)
			}
			rootPath, _ := getRootDir()
			t.save(rootPath)
		case "l", "list":
			if err := listTask(); err != nil {
				panic(err)
			}
		}
	} else {
		startTUI()
	}
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

	fmt.Print("tags (split by space): ")
	tagsStr, err := reader.ReadString('\n')
	t.tags = strings.Split(tagsStr, " ")

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

func getTask() ([]*task, error) {
	rootPath, err := getRootDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	var tasks []*task
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := loadTask(rootPath, entry.Name())
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func listTask() error {
	tasks, err := getTask()
	if err != nil {
		return err
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].priority > tasks[j].priority })
	for _, t := range tasks {
		status := "OPEN"
		if !t.done {
			status = "CLOSE"
		}
		tags := strings.Join(t.tags, ",")
		fmt.Printf("%s\t[%s]\t%d\t%s\t%s\n", t.id, status, t.priority, t.title, tags)
	}
	return nil
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

	inbody := false
	var bodyLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if inbody {
			bodyLines = append(bodyLines, line)
			continue
		}

		switch {
		case strings.HasPrefix(line, "# "):
			t.title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "- STATUS:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "- STATUS:"))
			t.done = v == "CLOSE"
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
		case strings.HasPrefix(line, "~ "):
			bodyLines = append(bodyLines, strings.TrimPrefix(line, "~ "))
			inbody = true
		}
		// description lines are intentionally not read further; we stop
		// caring about content once metadata is parsed since listTask
		// doesn't need it.
	}
	t.description = strings.Join(bodyLines, "\n")

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

func saveAll(tasks []*task) {
	root, err := getRootDir()
	if err != nil {
		return
	}
	for _, t := range tasks {
		t.save(root)
	}
}
