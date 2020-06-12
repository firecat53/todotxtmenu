package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/JamesClonk/go-todotxt"
)

var cmdPtr = flag.String("cmd", "dmenu", "Dmenu command to use (dmenu, rofi, wofi, etc)")
var todoPtr = flag.String("todo", "todo.txt", "Path to todo file")
var optsPtr = flag.String("opts", "", "Additional Rofi/Dmenu options")

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
		case out == "Delete Item":
			displayList, m = createMenu(&tasklist, true)
			out = display(displayList.String(), "SELECTED ITEM WILL BE DELETED")
			if err := tasklist.RemoveTaskById(m[out]); err != nil {
			}
		case out != "":
			t, _ := tasklist.GetTask(m[out])
			editItem(t)
		default:
			edit = false
		}
	}
}

func createMenu(tasklist *todotxt.TaskList, del bool) (strings.Builder, map[string]int) {
	// Sort tasklist by prioritized items first, then non-pri items by created
	// date. Don't display 'Add/Delete' options if del == true
	var displayList strings.Builder
	// Create map of task string->task Id's for reference
	m := make(map[string]int)
	if !del {
		displayList.WriteString("Add Item\nDelete Item\n")
	}
	prior := tasklist.Filter(func(t todotxt.Task) bool {
		return t.HasPriority()
	})
	if err := prior.Sort(todotxt.SORT_PRIORITY_ASC); err != nil {
		log.Fatal(err.Error())
	}
	displayList.WriteString(prior.String())
	for _, v := range *prior {
		m[v.String()] = v.Id
	}
	nonprior := tasklist.Filter(func(t todotxt.Task) bool {
		return !t.HasPriority()
	})
	if err := nonprior.Sort(todotxt.SORT_CREATED_DATE_DESC); err != nil {
		log.Fatal(err.Error())
	}
	displayList.WriteString(nonprior.String())
	for _, v := range *nonprior {
		m[v.String()] = v.Id
	}
	return displayList, m
}

func addItem(list *todotxt.TaskList) {
	// Add new todo item
	task := todotxt.NewTask()
	task = editItem(&task)
	if task.Todo != "" {
		list.AddTask(&task)
	}
}

func editItem(task *todotxt.Task) todotxt.Task {
	for edit := true; edit; {
		var displayList strings.Builder
		var tdd string
		if task.DueDate.IsZero() {
			tdd = ""
		} else {
			tdd = task.DueDate.Format("2006-01-02")
		}
		var comp string
		if task.Completed {
			comp = "\nRestore item (uncomplete)"
		} else {
			comp = "\nComplete item"
		}
		var tags, thd string
		for k, v := range task.AdditionalTags {
			if k == "t" {
				thd = v
				continue
			}
			tags = "\n" + tags + k + ": " + v
		}
		fmt.Fprint(&displayList,
			"Todo: "+task.Todo,
			"\nPriority: "+task.Priority,
			"\nProjects + (space separated): "+strings.Join(task.Projects, " "),
			"\nContexts @ (space separated): "+strings.Join(task.Contexts, " "),
			"\nDue date yyyy-mm-dd: "+tdd,
			"\nThreshold date yyyy-mm-dd: "+thd,
			tags,
			"\n",
			comp,
		)
		out := display(displayList.String(), task.String())
		switch {
		case strings.HasPrefix(out, "Todo"):
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
					task.AdditionalTags["t"] = ""
				} else {
					task.AdditionalTags["t"] = td.Format("2006-01-02")
				}
			}
		case strings.HasPrefix(out, "Complete item"):
			task.Completed = true
		case strings.HasPrefix(out, "Restore item"):
			task.Reopen()
		case strings.HasPrefix(out, "Delete item"):
			task.Completed = false
		case out != "":
			// TODO remove tag from AdditionalTags if empty
			for k, v := range task.AdditionalTags {
				if k == "t" {
					continue
				}
				if strings.HasPrefix(out, k) {
					task.AdditionalTags[k] = display(v, k)
					break
				}
			}
		default:
			edit = false
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
