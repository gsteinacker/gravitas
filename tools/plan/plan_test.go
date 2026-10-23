package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeTask(t *testing.T, dir string, task *Task) {
	t.Helper()
	if task.Path == "" {
		task.Path = filepath.Join(dir, task.ID+"-"+Slug(task.Title)+".md")
	}
	if err := os.WriteFile(task.Path, task.Format(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newTestPlan(t *testing.T, tasks ...*Task) *Plan {
	t.Helper()
	dir := t.TempDir()
	for _, task := range tasks {
		writeTask(t, dir, task)
	}
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func task(id, ms string, prio int, status Status, deps ...string) *Task {
	return &Task{ID: id, Title: "Task " + id, Status: status, Milestone: ms, Priority: prio, Estimate: "S", Depends: deps, Body: "\nbody\n"}
}

func ids(tasks []*Task) []string {
	var out []string
	for _, t := range tasks {
		out = append(out, t.ID)
	}
	return out
}

func TestParseFormatRoundTrip(t *testing.T) {
	in := "---\nid: T-007\ntitle: \"Barnes-Hut: Octree\"\nstatus: doing\nmilestone: M2\npriority: 1\nestimate: L\ndepends: [T-001, T-003]\n---\n\n## Ziel\n\nText\n"
	task, err := ParseTask("T-007-x.md", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if task.Title != "Barnes-Hut: Octree" || task.Status != StatusDoing || task.Priority != 1 ||
		!slices.Equal(task.Depends, []string{"T-001", "T-003"}) || task.Body != "\n## Ziel\n\nText\n" {
		t.Fatalf("unexpected task: %+v", task)
	}
	again, err := ParseTask("T-007-x.md", task.Format())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Format(), task.Format()) {
		t.Fatalf("format not stable:\n%s\nvs\n%s", again.Format(), task.Format())
	}
}

func TestParseErrors(t *testing.T) {
	for name, in := range map[string]string{
		"no start":      "id: T-001\n",
		"no end":        "---\nid: T-001\n",
		"bad status":    "---\nstatus: maybe\n---\n",
		"bad priority":  "---\npriority: high\n---\n",
		"unknown field": "---\nowner: me\n---\n",
	} {
		if _, err := ParseTask(name, []byte(in)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestReadyOrdersByMilestonePriorityID(t *testing.T) {
	p := newTestPlan(t,
		task("T-001", "M0", 1, StatusDone),
		task("T-002", "M10", 1, StatusTodo),
		task("T-003", "M1", 2, StatusTodo, "T-001"),
		task("T-004", "M1", 1, StatusTodo, "T-005"),
		task("T-005", "M1", 3, StatusTodo),
		task("T-006", "M1", 2, StatusTodo),
	)
	if got, want := ids(p.Ready()), []string{"T-003", "T-006", "T-005", "T-002"}; !slices.Equal(got, want) {
		t.Fatalf("Ready() = %v, want %v", got, want)
	}
	if got := p.NextID(); got != "T-007" {
		t.Fatalf("NextID() = %s", got)
	}
}

func TestCheck(t *testing.T) {
	ok := newTestPlan(t, task("T-001", "M0", 1, StatusDone), task("T-002", "M0", 2, StatusTodo, "T-001"))
	if errs := ok.Check(); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	bad := newTestPlan(t,
		task("T-001", "M0", 1, StatusTodo, "T-002"),
		task("T-002", "M0", 1, StatusTodo, "T-001"),
		task("T-003", "MX", 5, StatusDone, "T-001", "T-099"),
	)
	var msgs []string
	for _, e := range bad.Check() {
		msgs = append(msgs, e.Error())
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"dependency cycle", "unknown task T-099", "invalid milestone", "out of range", "done but depends"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in:\n%s", want, all)
		}
	}
}

func TestSetAndNewCommands(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, task("T-001", "M0", 1, StatusTodo))
	var out bytes.Buffer
	if err := run([]string{"-dir", dir, "new", "-m", "M1", "-d", "T-001", "Schwarzes", "Loch", "platzieren"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(out.String()), "T-002-schwarzes-loch-platzieren.md") {
		t.Fatalf("unexpected path %q", out.String())
	}
	if err := run([]string{"-dir", dir, "set", "T-002", "doing"}, &out); err == nil {
		t.Fatal("expected refusal because T-001 is not done")
	}
	for _, args := range [][]string{{"set", "T-001", "done"}, {"set", "T-002", "doing"}, {"check"}} {
		if err := run(append([]string{"-dir", dir}, args...), &out); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Get("T-002").Status != StatusDoing {
		t.Fatalf("T-002 status = %s", p.Get("T-002").Status)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Barnes-Hut Octree (parallel)": "barnes-hut-octree-parallel",
		"Größe & Übergänge":            "groesse-uebergaenge",
		"  --x--  ":                    "x",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
