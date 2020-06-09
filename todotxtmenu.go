package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/JamesClonk/go-todotxt"
)

var dmenuOpts = []string{
	"rofi",
	"-i",
	"-dmenu",
	"-multi-select",
	"-theme",
	"keepmenu",
}

func main() {
	tasklist, err := todotxt.LoadFromFilename("todo.txt")
	if err != nil {
		log.Fatal(err)
	}
	edit := true
	for edit {
		displayList := createMenu(&tasklist)
		out := display(displayList.String(), dmenuOpts)
		switch {
		case out == "Add Item":
			addItem(&tasklist)
		default:
			edit = false
			fmt.Print(tasklist)
		}
	}
}

func createMenu(tasklist *todotxt.TaskList) strings.Builder {
	var displayList strings.Builder
	displayList.WriteString("Add Item\n")
	prior := tasklist.Filter(func(t todotxt.Task) bool {
		return t.HasPriority()
	})
	if err := prior.Sort(todotxt.SORT_PRIORITY_ASC); err != nil {
		log.Fatal(err)
	}
	displayList.WriteString(prior.String())
	nonprior := tasklist.Filter(func(t todotxt.Task) bool {
		return !t.HasPriority()
	})
	if err := nonprior.Sort(todotxt.SORT_CREATED_DATE_DESC); err != nil {
		log.Fatal(err)
	}
	displayList.WriteString(nonprior.String())
	return displayList
}

func addItem(list *todotxt.TaskList) {
	// Add new todo item
	task := todotxt.NewTask()
	edit := true
	for edit {
		var displayList strings.Builder
		var tdd string
		if task.DueDate.IsZero() {
			tdd = ""
		} else {
			tdd = task.DueDate.Format("2006-01-02")
		}
		fmt.Fprint(&displayList,
			"Todo: "+task.Todo,
			"\nPriority: "+task.Priority,
			"\nProjects + (space separated): "+strings.Join(task.Projects, " "),
			"\nContexts @ (space separated): "+strings.Join(task.Contexts, " "),
			"\nDue date yyyy-mm-dd: "+tdd,
			"\nThreshold yyyy-mm-dd: ",
		)
		out := display(displayList.String(), dmenuOpts)
		switch {
		case strings.HasPrefix(out, "Todo"):
			task.Todo = display(task.Todo, append(dmenuOpts, "-p", "Todo Title: "))
		case strings.HasPrefix(out, "Priority"):
			task.Priority = strings.ToUpper(display(task.Priority, append(dmenuOpts, "-p", "Priority:")))
		case strings.HasPrefix(out, "Projects"):
			p := display(strings.Join(task.Projects, " "), append(dmenuOpts, "-p", "Projects (+):"))
			task.Projects = strings.Split(p, " ")
		case strings.HasPrefix(out, "Contexts"):
			p := display(strings.Join(task.Contexts, " "), append(dmenuOpts, "-p", "Contexts (@):"))
			task.Contexts = strings.Split(p, " ")
		case strings.HasPrefix(out, "Due date"):
			t := display(tdd, append(dmenuOpts, "-p", "Due Date (yyyy-mm-dd):"))
			td, err := time.Parse("2006-01-02", t)
			if err != nil {
				display("", append(dmenuOpts, "-p", "Bad date format"))
				break
			} else {
				task.DueDate = td
			}
		default:
			edit = false
			if task.Todo != "" {
				list.AddTask(&task)
			}
		}
	}
}

// func todoTitle(list *todotxt.TaskList) *todotxt.TaskList {
// Prompt for new Todo item
// 	var displayList strings.Builder
// 	item := display(displayList, dmenuOpts)
// 	if item != "" {
// 		task, err := todotxt.ParseTask(item)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		return task
// 	}
// }

func display(list string, opts []string) (result string) {
	// Displays list in dmenu, returns selection
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd := exec.Command(opts[0], opts[1:]...)
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	cmd.Stdin = strings.NewReader(list)
	err := cmd.Run()
	if err != nil {
		if outErr.String() != "" {
			log.Fatal(outErr)
		} else {
			return
		}
		log.Fatal(err)
	}
	result = strings.TrimRight(out.String(), "\n")
	return
}
