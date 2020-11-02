package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JamesClonk/go-todotxt"
)

var archPtr = flag.Bool("archive", true, "Move completed items to done.txt")
var cmdPtr = flag.String("cmd", "dmenu", "Dmenu command to use (dmenu, rofi, wofi, etc)")
var optsPtr = flag.String("opts", "", "Additional Rofi/Dmenu options")
var thresholdPtr = flag.Bool("threshold", false, "Hide items before their threshold date")
var todoPtr = flag.String("todo", "todo.txt", "Path to todo file")

func main() {
	flag.Parse()
	tasklist, err := todotxt.LoadFromFilename(*todoPtr)
	if err != nil {
		log.Fatal(err.Error())
	}
	for edit := true; edit; {
		displayList, m := createMenu(&tasklist, false)
		out := display(displayList.String(), *todoPtr)
		switch {
		case out == "Add Item":
			addItem(&tasklist)
		case out != "":
			t, _ := tasklist.GetTask(m[out])
			editItem(t, &tasklist)
		default:
			edit = false
		}
	}
	if *archPtr {
		archiveDone(&tasklist)
	}
	if err := todotxt.WriteToFilename(&tasklist, *todoPtr); err != nil {
		log.Fatal(err.Error())
	}
}

func archiveDone(tasklist *todotxt.TaskList) {
	// Archive completed items to <path/to/done.txt>
	path := filepath.Join(filepath.Dir(*todoPtr), "done.txt")
	// create done.txt if necessary
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err.Error())
	}
	file.Close()
	alldone, err := todotxt.LoadFromFilename(path)
	if err != nil {
		log.Fatal(err.Error())
	}
	done := tasklist.Filter(func(t todotxt.Task) bool {
		return t.Completed
	})
	for _, t := range *done {
		if err := tasklist.RemoveTaskById(t.Id); err != nil {
			fmt.Printf("Unable to remove task #%s", t.String())
		}
	}
	x := append(alldone, *done...)
	if err := todotxt.WriteToFilename(&x, path); err != nil {
		log.Fatal(err.Error())
	}
}

func createMenu(tasklist *todotxt.TaskList, del bool) (strings.Builder, map[string]int) {
	// Sort tasklist by prioritized items first, then non-pri items, then
	// completed items by created date. Don't display 'Add/Delete' options if
	// del == true
	var displayList strings.Builder
	if !del {
		displayList.WriteString("Add Item\n")
	}
	prior := tasklist.Filter(func(t todotxt.Task) bool {
		return t.HasPriority() && !t.Completed && checkThreshold(t)
	})
	if err := prior.Sort(todotxt.SORT_PRIORITY_ASC); err != nil {
		log.Fatal(err.Error())
	}
	nonprior := tasklist.Filter(func(t todotxt.Task) bool {
		return !t.HasPriority() && !t.Completed && checkThreshold(t)
	})
	if err := nonprior.Sort(todotxt.SORT_CREATED_DATE_DESC); err != nil {
		log.Fatal(err.Error())
	}
	done := tasklist.Filter(func(t todotxt.Task) bool {
		return t.Completed
	})
	if err := done.Sort(todotxt.SORT_CREATED_DATE_DESC); err != nil {
		log.Fatal(err.Error())
	}
	displayList.WriteString(prior.String() + nonprior.String() + done.String())
	// Create map of task string->task Id's for reference
	m := make(map[string]int)
	for _, v := range append(*prior, append(*nonprior, *done...)...) {
		m[v.String()] = v.Id
	}
	return displayList, m
}

func checkThreshold(t todotxt.Task) bool {
	// Return true if threshold date is before now, date doesn't parse
	// correctly, or -threshold isn't set. False if td in the future
	if !*thresholdPtr {
		return true
	}
	if d, ok := t.AdditionalTags["t"]; ok {
		td, err := time.Parse("2006-01-02", d)
		if err != nil && d != "" {
			return true
		}
		return td.Before(time.Now())
	}
	return true
}

func addItem(list *todotxt.TaskList) {
	// Add new todo item
	t := todotxt.NewTask()
	t.Todo = display(t.Todo, "Todo Title: ")
	task := &t
	if t.Todo != "" {
		task, _ = todotxt.ParseTask(t.String())
	}
	task1 := editItem(task, list)
	if task1.Todo != "" {
		list.AddTask(&task1)
	}
}

func getAllProjCont(tasklist *todotxt.TaskList) (projects []string, contexts []string) {
	// Return arrays of all projects and all contexts
	for _, v := range *tasklist {
		projects = append(v.Projects, projects...)
		contexts = append(v.Contexts, contexts...)
	}
	p, c := dedupeList(projects), dedupeList(contexts)
	sort.Strings(p)
	sort.Strings(c)
	return p, c
}

func dedupeList(list []string) (result []string) {
	// Given a list of strings, remove duplicates and return
	flag := make(map[string]bool)
	for _, name := range list {
		if flag[name] == false {
			flag[name] = true
			result = append(result, name)
		}
	}
	return
}

func editItem(task *todotxt.Task, tasklist *todotxt.TaskList) todotxt.Task {
	projects, contexts := getAllProjCont(tasklist)
	taskOrig := task.String()
	t, _ := todotxt.ParseTask(taskOrig)
	for edit := true; edit; {
		// Initialize AdditionalTags if not already
		if len(t.AdditionalTags) == 0 {
			t.AdditionalTags = make(map[string]string)
		}
		var displayList strings.Builder
		var tdd string
		if t.DueDate.IsZero() {
			tdd = ""
		} else {
			tdd = t.DueDate.Format("2006-01-02")
		}
		var comp string
		if len(t.Todo) == 0 {
			comp = ""
		} else if t.Completed {
			comp = "Restore item (uncomplete)\n\n"
		} else {
			comp = "Complete item\n\n"
		}
		var tags, thd string
		for k, v := range t.AdditionalTags {
			if k == "t" {
				// Handle threshold (t:) tag
				thd = v
				continue
			}
			tags = tags + "\n" + k + ": " + v
		}
		fmt.Fprint(&displayList,
			"Save item\n",
			comp,
			"Title: "+t.Todo,
			"\nPriority: "+t.Priority,
			"\nContexts @ (space separated): "+strings.Join(t.Contexts, " "),
			"\nProjects + (space separated): "+strings.Join(t.Projects, " "),
			"\nDue date yyyy-mm-dd: "+tdd,
			"\nThreshold date yyyy-mm-dd: "+thd,
			tags,
			"\n\nDelete item",
		)
		out := display(displayList.String(), t.String())
		switch {
		case out == "Save item":
			edit = false
			*task = *t
		case strings.HasPrefix(out, "Title"):
			t.Todo = display(t.Todo, "Todo Title: ")
		case strings.HasPrefix(out, "Priority"):
			// Convert this to []rune to allow comparison to 'A' and 'Z' instead
			// of adding regex or unicode dependency
			p := []rune(strings.ToUpper(display(t.Priority, "Priority:")))
			if len(p) > 1 || (len(p) > 0 && (p[0] < 'A' || p[0] > 'Z')) {
				display("", "Priority must be single letter A-Z")
				break
			}
			t.Priority = string(p)
		case strings.HasPrefix(out, "Projects"):
			prj := strings.Join(t.Projects, " ") + "\n\n" + strings.Join(projects, "\n")
			p := display(prj, "Projects (+):")
			if p != "" {
				t.Projects = strings.Split(p, " ")
			} else {
				t.Projects = []string{}
			}
		case strings.HasPrefix(out, "Contexts"):
			cont := strings.Join(t.Contexts, " ") + "\n\n" + strings.Join(contexts, "\n")
			c := display(cont, "Contexts (+):")
			if c != "" {
				t.Contexts = strings.Split(c, " ")
			} else {
				t.Contexts = []string{}
			}
		case strings.HasPrefix(out, "Due date"):
			d := display(tdd, "Due Date (yyyy-mm-dd):")
			td, err := time.Parse("2006-01-02", d)
			if err != nil && d != "" {
				display("", "Bad date format. Should be yyyy-mm-dd.")
				break
			} else {
				t.DueDate = td
			}
		case strings.HasPrefix(out, "Threshold"):
			d := display(thd, "Threshold Date (yyyy-mm-dd):")
			td, err := time.Parse("2006-01-02", d)
			if err != nil && d != "" {
				display("", "Bad date format. Should be yyyy-mm-dd.")
				break
			} else {
				// Threshold date is an additional tag and stored as a string
				// not as a time object
				if td.IsZero() {
					delete(t.AdditionalTags, "t")
				} else {
					t.AdditionalTags["t"] = td.Format("2006-01-02")
				}
			}
		case strings.HasPrefix(out, "Complete item"):
			t.Completed = true
		case strings.HasPrefix(out, "Restore item"):
			t.Reopen()
		case strings.HasPrefix(out, "Delete item"):
			if err := tasklist.RemoveTaskById(task.Id); err != nil {
				// new tasks don't have an Id yet
				t.Todo = ""
			}
			edit = false
		case out != "":
			for k, v := range t.AdditionalTags {
				if k == "t" {
					continue
				}
				if strings.HasPrefix(out, k) {
					if d := display(v, k); d == "" {
						delete(t.AdditionalTags, k)
					} else {
						t.AdditionalTags[k] = d
					}
					break
				}
			}
		default:
			edit = false
		}
		if t.Todo != "" {
			t, _ = todotxt.ParseTask(t.String())
		}
	}
	return *task
}

func display(list string, title string) (result string) {
	// Displays list in dmenu, returns selection
	var out, outErr bytes.Buffer
	flag.Parse()
	opts := strings.Split(*optsPtr, " ")
	o := []string{"-i", "-p", title}
	if *cmdPtr == "rofi" {
		o = []string{"-i", "-dmenu", "-p", title}
	}
	// Remove empty "" from dmenu args that would cause a dmenu error
	if opts[0] != "" {
		opts = append(o, opts...)
	} else {
		opts = o
	}
	cmd := exec.Command(*cmdPtr, opts...)
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	cmd.Stdin = strings.NewReader(list)
	err := cmd.Run()
	if err != nil {
		if outErr.String() != "" {
			log.Fatal(outErr.String())
		} else {
			// Skip this error when hitting Esc to go back to previous menu
			if err.Error() == "exit status 1" {
				return
			}
			log.Fatal(err.Error())
		}
		log.Fatal(err.Error())
	}
	result = strings.TrimRight(out.String(), "\n")
	return
}
