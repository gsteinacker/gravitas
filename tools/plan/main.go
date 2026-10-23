// Command plan manages the file-based backlog in planning/tasks.
//
//	plan list   [-s status] [-m milestone]   tabular overview
//	plan board                               progress per milestone
//	plan next                                tasks in progress and the next ready task
//	plan show   <id>                         print a task file
//	plan set    [-force] <id> <status>       change a task's status
//	plan new    [-m M0] [-p 2] [-e M] [-d T-001,T-002] <title...>
//	plan check                               validate the backlog (exit 1 on problems)
//
// The backlog directory defaults to planning/tasks below the module root and can be
// overridden with -dir or $PLAN_DIR.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "plan:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	global := flag.NewFlagSet("plan", flag.ContinueOnError)
	dir := global.String("dir", os.Getenv("PLAN_DIR"), "backlog directory")
	if err := global.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		*dir = filepath.Join(moduleRoot(), "planning", "tasks")
	}
	if global.NArg() == 0 {
		return errors.New("missing command: list | board | next | show | set | new | check")
	}
	cmd, rest := global.Arg(0), global.Args()[1:]

	p, loadErr := Load(*dir)
	if loadErr != nil && cmd != "check" {
		return loadErr
	}
	switch cmd {
	case "list":
		return cmdList(p, rest, out)
	case "board":
		return cmdBoard(p, out)
	case "next":
		return cmdNext(p, out)
	case "show":
		return cmdShow(p, rest, out)
	case "set":
		return cmdSet(p, rest, out)
	case "new":
		return cmdNew(p, rest, out)
	case "check":
		return cmdCheck(p, loadErr, out)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func cmdList(p *Plan, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	status := fs.String("s", "", "filter by status")
	ms := fs.String("m", "", "filter by milestone")
	if err := fs.Parse(args); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tMS\tPRIO\tEST\tTITLE")
	for _, t := range p.Tasks {
		if *status != "" && string(t.Status) != *status || *ms != "" && t.Milestone != *ms {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n", t.ID, t.Status, t.Milestone, t.Priority, t.Estimate, t.Title)
	}
	return tw.Flush()
}

var marker = map[Status]string{StatusDone: "[x]", StatusDoing: "[>]", StatusBlocked: "[!]", StatusTodo: "[ ]"}

func cmdBoard(p *Plan, out io.Writer) error {
	for _, ms := range p.Milestones() {
		var tasks []*Task
		done := 0
		for _, t := range p.Tasks {
			if t.Milestone == ms {
				tasks = append(tasks, t)
				if t.Status == StatusDone {
					done++
				}
			}
		}
		fmt.Fprintf(out, "%s  %s  %d/%d\n", ms, progressBar(done, len(tasks), 20), done, len(tasks))
		for _, t := range tasks {
			fmt.Fprintf(out, "  %s %s %s\n", marker[t.Status], t.ID, t.Title)
		}
		fmt.Fprintln(out)
	}
	return nil
}

func progressBar(done, total, width int) string {
	filled := 0
	if total > 0 {
		filled = done * width / total
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "]"
}

func cmdNext(p *Plan, out io.Writer) error {
	if doing := p.WithStatus(StatusDoing); len(doing) > 0 {
		fmt.Fprintln(out, "In progress:")
		for _, t := range doing {
			fmt.Fprintf(out, "  %s %s\n", t.ID, t.Title)
		}
		fmt.Fprintln(out)
	}
	ready := p.Ready()
	if len(ready) == 0 {
		fmt.Fprintln(out, "No ready tasks.")
		return nil
	}
	if len(ready) > 1 {
		fmt.Fprintln(out, "Also ready:")
		for _, t := range ready[1:min(len(ready), 6)] {
			fmt.Fprintf(out, "  %s %s (%s, P%d)\n", t.ID, t.Title, t.Milestone, t.Priority)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "Next: %s\n\n", ready[0].Path)
	_, err := out.Write(ready[0].Format())
	return err
}

func cmdShow(p *Plan, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: plan show <id>")
	}
	t := p.Get(args[0])
	if t == nil {
		return fmt.Errorf("unknown task %s", args[0])
	}
	_, err := out.Write(t.Format())
	return err
}

func cmdSet(p *Plan, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	force := fs.Bool("force", false, "ignore unfinished dependencies")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: plan set [-force] <id> <status>")
	}
	t := p.Get(fs.Arg(0))
	if t == nil {
		return fmt.Errorf("unknown task %s", fs.Arg(0))
	}
	status, err := parseStatus(fs.Arg(1))
	if err != nil {
		return err
	}
	if open := p.Unfinished(t); len(open) > 0 && !*force && (status == StatusDoing || status == StatusDone) {
		return fmt.Errorf("%s depends on unfinished %v (use -force to override)", t.ID, open)
	}
	t.Status = status
	if err := os.WriteFile(t.Path, t.Format(), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s -> %s\n", t.ID, status)
	return nil
}

const newTaskBody = `
## Ziel

TODO: Was soll erreicht werden und warum?

## Akzeptanzkriterien

- [ ] TODO

## Notizen
`

func cmdNew(p *Plan, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	ms := fs.String("m", "M0", "milestone")
	prio := fs.Int("p", 2, "priority 1..3")
	est := fs.String("e", "M", "estimate S|M|L")
	deps := fs.String("d", "", "comma-separated dependencies")
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.Join(fs.Args(), " ")
	if title == "" {
		return errors.New("usage: plan new [-m M0] [-p 2] [-e M] [-d T-001,T-002] <title...>")
	}
	t := &Task{
		ID: p.NextID(), Title: title, Status: StatusTodo, Milestone: *ms,
		Priority: *prio, Estimate: *est, Depends: parseList(*deps), Body: newTaskBody,
	}
	t.Path = filepath.Join(p.Dir, t.ID+"-"+Slug(title)+".md")
	p.add(t)
	if errs := p.Check(); len(errs) > 0 {
		return errors.Join(errs...)
	}
	if err := os.WriteFile(t.Path, t.Format(), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(out, t.Path)
	return nil
}

func cmdCheck(p *Plan, loadErr error, out io.Writer) error {
	errs := p.Check()
	if loadErr != nil {
		errs = append([]error{loadErr}, errs...)
	}
	for _, e := range errs {
		fmt.Fprintln(out, e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d problem(s)", len(errs))
	}
	fmt.Fprintf(out, "ok: %d tasks\n", len(p.Tasks))
	return nil
}

// moduleRoot walks up from the working directory to the nearest go.mod.
func moduleRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if filepath.Dir(dir) == dir {
			return wd
		}
	}
}
