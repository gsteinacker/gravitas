package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Status is the lifecycle state of a task.
type Status string

const (
	StatusTodo    Status = "todo"
	StatusDoing   Status = "doing"
	StatusBlocked Status = "blocked"
	StatusDone    Status = "done"
)

var validStatuses = []Status{StatusTodo, StatusDoing, StatusBlocked, StatusDone}

func parseStatus(s string) (Status, error) {
	for _, v := range validStatuses {
		if string(v) == s {
			return v, nil
		}
	}
	return "", fmt.Errorf("invalid status %q (want one of %v)", s, validStatuses)
}

// Task is one backlog item, stored as a Markdown file with a small frontmatter block.
type Task struct {
	ID        string
	Title     string
	Status    Status
	Milestone string
	Priority  int    // 1 = high, 3 = low
	Estimate  string // S, M, L
	Depends   []string

	Path string // file the task was loaded from
	Body string // Markdown after the frontmatter
}

const frontDelim = "---\n"

// ParseTask parses a task file. The frontmatter is a strict subset of YAML:
// one "key: value" per line, lists written as [a, b].
func ParseTask(path string, data []byte) (*Task, error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, frontDelim) {
		return nil, fmt.Errorf("%s: missing frontmatter start %q", path, "---")
	}
	rest := s[len(frontDelim):]
	end := strings.Index(rest, "\n"+frontDelim)
	if end < 0 {
		return nil, fmt.Errorf("%s: missing frontmatter end %q", path, "---")
	}
	t := &Task{Path: path, Body: rest[end+1+len(frontDelim):]}
	for i, line := range strings.Split(rest[:end], "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("%s:%d: expected \"key: value\"", path, i+2)
		}
		key, val = strings.TrimSpace(key), unquote(strings.TrimSpace(val))
		var err error
		switch key {
		case "id":
			t.ID = val
		case "title":
			t.Title = val
		case "status":
			t.Status, err = parseStatus(val)
		case "milestone":
			t.Milestone = val
		case "priority":
			t.Priority, err = strconv.Atoi(val)
		case "estimate":
			t.Estimate = val
		case "depends":
			t.Depends = parseList(val)
		default:
			err = fmt.Errorf("unknown field %q", key)
		}
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, i+2, err)
		}
	}
	return t, nil
}

// Format renders the task back into its file representation with a canonical field order.
func (t *Task) Format() []byte {
	var b strings.Builder
	b.WriteString(frontDelim)
	fmt.Fprintf(&b, "id: %s\n", t.ID)
	fmt.Fprintf(&b, "title: %s\n", t.Title)
	fmt.Fprintf(&b, "status: %s\n", t.Status)
	fmt.Fprintf(&b, "milestone: %s\n", t.Milestone)
	fmt.Fprintf(&b, "priority: %d\n", t.Priority)
	fmt.Fprintf(&b, "estimate: %s\n", t.Estimate)
	fmt.Fprintf(&b, "depends: [%s]\n", strings.Join(t.Depends, ", "))
	b.WriteString(frontDelim)
	b.WriteString(t.Body)
	return []byte(b.String())
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func parseList(s string) []string {
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	var out []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
