package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	idPattern        = regexp.MustCompile(`^T-(\d{3,})$`)
	milestonePattern = regexp.MustCompile(`^M(\d+)$`)
)

// Plan is the set of all tasks in the backlog directory.
type Plan struct {
	Dir   string
	Tasks []*Task // sorted by milestone, priority, id
	byID  map[string]*Task
}

// Load reads every *.md file in dir as a task.
func Load(dir string) (*Plan, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	p := &Plan{Dir: dir, byID: map[string]*Task{}}
	var errs []error
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		t, err := ParseTask(path, data)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		p.add(t)
	}
	p.sort()
	return p, errors.Join(errs...)
}

func (p *Plan) add(t *Task) {
	p.Tasks = append(p.Tasks, t)
	if _, dup := p.byID[t.ID]; !dup {
		p.byID[t.ID] = t
	}
}

func (p *Plan) sort() {
	slices.SortStableFunc(p.Tasks, func(a, b *Task) int {
		if c := milestoneNum(a.Milestone) - milestoneNum(b.Milestone); c != 0 {
			return c
		}
		if c := a.Priority - b.Priority; c != 0 {
			return c
		}
		return idNum(a.ID) - idNum(b.ID)
	})
}

// Get returns the task with the given id, or nil.
func (p *Plan) Get(id string) *Task { return p.byID[id] }

// Unfinished returns the dependencies of t that are not done (including unknown ones).
func (p *Plan) Unfinished(t *Task) []string {
	var out []string
	for _, d := range t.Depends {
		if dt := p.byID[d]; dt == nil || dt.Status != StatusDone {
			out = append(out, d)
		}
	}
	return out
}

// Ready returns all todo tasks whose dependencies are done, in plan order.
func (p *Plan) Ready() []*Task {
	var out []*Task
	for _, t := range p.Tasks {
		if t.Status == StatusTodo && len(p.Unfinished(t)) == 0 {
			out = append(out, t)
		}
	}
	return out
}

// WithStatus returns all tasks in the given state, in plan order.
func (p *Plan) WithStatus(s Status) []*Task {
	var out []*Task
	for _, t := range p.Tasks {
		if t.Status == s {
			out = append(out, t)
		}
	}
	return out
}

// NextID returns the id a new task should get.
func (p *Plan) NextID() string {
	maxN := 0
	for _, t := range p.Tasks {
		maxN = max(maxN, idNum(t.ID))
	}
	return fmt.Sprintf("T-%03d", maxN+1)
}

// Milestones returns the distinct milestones in order.
func (p *Plan) Milestones() []string {
	var out []string
	for _, t := range p.Tasks {
		if !slices.Contains(out, t.Milestone) {
			out = append(out, t.Milestone)
		}
	}
	return out
}

// Check validates the backlog and returns one error per problem found.
func (p *Plan) Check() []error {
	var errs []error
	fail := func(t *Task, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", filepath.Base(t.Path), fmt.Sprintf(format, args...)))
	}
	seen := map[string]bool{}
	for _, t := range p.Tasks {
		if !idPattern.MatchString(t.ID) {
			fail(t, "invalid id %q (want T-NNN)", t.ID)
		}
		if seen[t.ID] {
			fail(t, "duplicate id %s", t.ID)
		}
		seen[t.ID] = true
		if base := filepath.Base(t.Path); base != t.ID+".md" && !strings.HasPrefix(base, t.ID+"-") {
			fail(t, "file name must start with %q", t.ID+"-")
		}
		if t.Title == "" {
			fail(t, "missing title")
		}
		if t.Status == "" {
			fail(t, "missing status")
		}
		if !milestonePattern.MatchString(t.Milestone) {
			fail(t, "invalid milestone %q (want MN)", t.Milestone)
		}
		if t.Priority < 1 || t.Priority > 3 {
			fail(t, "priority %d out of range 1..3", t.Priority)
		}
		if !slices.Contains([]string{"", "S", "M", "L"}, t.Estimate) {
			fail(t, "invalid estimate %q (want S, M or L)", t.Estimate)
		}
		for _, d := range t.Depends {
			switch {
			case d == t.ID:
				fail(t, "depends on itself")
			case p.byID[d] == nil:
				fail(t, "depends on unknown task %s", d)
			}
		}
		if t.Status == StatusDone {
			if open := p.Unfinished(t); len(open) > 0 {
				fail(t, "done but depends on unfinished %v", open)
			}
		}
	}
	if cycle := p.findCycle(); cycle != nil {
		errs = append(errs, fmt.Errorf("dependency cycle: %s", strings.Join(cycle, " -> ")))
	}
	return errs
}

// findCycle returns one dependency cycle as a path of ids, or nil.
func (p *Plan) findCycle() []string {
	const (
		unvisited = iota
		inProgress
		finished
	)
	state := map[string]int{}
	var stack []string
	var visit func(id string) []string
	visit = func(id string) []string {
		switch state[id] {
		case inProgress:
			start := slices.Index(stack, id)
			return append(slices.Clone(stack[start:]), id)
		case finished:
			return nil
		}
		state[id] = inProgress
		stack = append(stack, id)
		if t := p.byID[id]; t != nil {
			for _, d := range t.Depends {
				if c := visit(d); c != nil {
					return c
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = finished
		return nil
	}
	for _, t := range p.Tasks {
		if c := visit(t.ID); c != nil {
			return c
		}
	}
	return nil
}

func idNum(id string) int {
	m := idPattern.FindStringSubmatch(id)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func milestoneNum(ms string) int {
	m := milestonePattern.FindStringSubmatch(ms)
	if m == nil {
		return 1 << 30 // unknown milestones sort last
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

var umlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")

// Slug turns a title into a short file-name-safe string.
func Slug(title string) string {
	s := umlauts.Replace(strings.ToLower(title))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimSuffix(b.String(), "-")
	if len(out) > 40 {
		out = strings.TrimSuffix(out[:40], "-")
	}
	return out
}
