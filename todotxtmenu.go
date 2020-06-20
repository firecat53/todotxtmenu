package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JamesClonk/go-todotxt"
)

var archPtr = flag.Bool("archive", true, "Move completed items to done.txt")
var cmdPtr = flag.String("cmd", "dmenu", "Dmenu command to use (dmenu, rofi, wofi, etc)")
var optsPtr = flag.String("opts", "", "Additional Rofi/Dmenu options")
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
		return t.HasPriority() && !t.Completed
	})
	if err := prior.Sort(todotxt.SORT_PRIORITY_ASC); err != nil {
		log.Fatal(err.Error())
	}
	nonprior := tasklist.Filter(func(t todotxt.Task) bool {
		return !t.HasPriority() && !t.Completed
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

func addItem(list *todotxt.TaskList) {
	// Add new todo item
	task := todotxt.NewTask()
	task = editItem(&task, list)
	if task.Todo != "" {
		list.AddTask(&task)
	}
}

func editItem(task *todotxt.Task, tasklist *todotxt.TaskList) todotxt.Task {
	for edit := true; edit; {
		// Initialize AdditionalTags if not already
		if len(task.AdditionalTags) == 0 {
			task.AdditionalTags = make(map[string]string)
		}
		var displayList strings.Builder
		var tdd string
		if task.DueDate.IsZero() {
			tdd = ""
		} else {
			tdd = task.DueDate.Format("2006-01-02")
		}
		var comp string
		if len(task.Todo) == 0 {
			comp = ""
		} else if task.Completed {
			comp = "Restore item (uncomplete)\n\n"
		} else {
			comp = "Complete item\n\n"
		}
		var tags, thd string
		for k, v := range task.AdditionalTags {
			if k == "t" {
				// Handle threshold (t:) tag
				thd = v
				continue
			}
			tags = tags + "\n" + k + ": " + v
		}
		fmt.Fprint(&displayList,
			comp,
			"Title: "+task.Todo,
			"\nPriority: "+task.Priority,
			"\nContexts @ (space separated): "+strings.Join(task.Contexts, " "),
			"\nProjects + (space separated): "+strings.Join(task.Projects, " "),
			"\nDue date yyyy-mm-dd: "+tdd,
			"\nThreshold date yyyy-mm-dd: "+thd,
			tags,
			"\n\nDelete item",
		)
		out := display(displayList.String(), task.String())
		switch {
		case strings.HasPrefix(out, "Title"):
			task.Todo = display(task.Todo, "Todo Title: ")
		case strings.HasPrefix(out, "Priority"):
			// Convert this to []rune to allow comparison to 'A' and 'Z' instead
			// of adding regex or unicode dependency
			p := []rune(strings.ToUpper(display(task.Priority, "Priority:")))
			if len(p) > 1 || (len(p) > 0 && (p[0] < 'A' || p[0] > 'Z')) {
				display("", "Priority must be single letter A-Z")
				break
			}
			task.Priority = string(p)
		case strings.HasPrefix(out, "Projects"):
			p := display(strings.Join(task.Projects, " "), "Projects (+):")
			task.Projects = strings.Split(p, " ")
		case strings.HasPrefix(out, "Contexts"):
			p := display(strings.Join(task.Contexts, " "), "Contexts (@):")
			task.Contexts = strings.Split(p, " ")
		case strings.HasPrefix(out, "Due date"):
			t := display(tdd, "Due Date (yyyy-mm-dd):")
			td, err := time.Parse("2006-01-02", t)
			if err != nil && t != "" {
				display("", "Bad date format. Should be yyyy-mm-dd.")
				break
			} else {
				task.DueDate = td
			}
		case strings.HasPrefix(out, "Threshold"):
			t := display(thd, "Threshold Date (yyyy-mm-dd):")
			td, err := time.Parse("2006-01-02", t)
			if err != nil && t != "" {
				display("", "Bad date format. Should be yyyy-mm-dd.")
				break
			} else {
				// Threshold date is an additional tag and stored as a string
				// not as a time object
				if td.IsZero() {
					delete(task.AdditionalTags, "t")
				} else {
					task.AdditionalTags["t"] = td.Format("2006-01-02")
				}
			}
		case strings.HasPrefix(out, "Complete item"):
			task.Completed = true
		case strings.HasPrefix(out, "Restore item"):
			task.Reopen()
		case strings.HasPrefix(out, "Delete item"):
			if err := tasklist.RemoveTaskById(task.Id); err != nil {
				// new tasks don't have an Id yet
				task.Todo = ""
			}
			edit = false
		case out != "":
			for k, v := range task.AdditionalTags {
				if k == "t" {
					continue
				}
				if strings.HasPrefix(out, k) {
					if t := display(v, k); t == "" {
						delete(task.AdditionalTags, k)
					} else {
						task.AdditionalTags[k] = t
					}
					break
				}
			}
		default:
			edit = false
		}
		if task.Todo != "" {
			task, _ = todotxt.ParseTask(task.String())
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
