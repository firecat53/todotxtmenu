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
		displayList, m := createMenu(&tasklist)
		out := display(displayList.String(), dmenuOpts)
		switch {
		case out == "Add Item":
			addItem(&tasklist)
		case out != "":
			id := m[out]
			t, _ := tasklist.GetTask(id)
			editItem(t)
		default:
			edit = false
		}
	}
}

func createMenu(tasklist *todotxt.TaskList) (strings.Builder, map[string]int) {
	var displayList strings.Builder
	m := make(map[string]int)
	displayList.WriteString("Add Item\n")
	prior := tasklist.Filter(func(t todotxt.Task) bool {
		return t.HasPriority()
	})
	if err := prior.Sort(todotxt.SORT_PRIORITY_ASC); err != nil {
		log.Fatal(err)
	}
	displayList.WriteString(prior.String())
	for _, v := range *prior {
		m[v.String()] = v.Id
	}
	nonprior := tasklist.Filter(func(t todotxt.Task) bool {
		return !t.HasPriority()
	})
	if err := nonprior.Sort(todotxt.SORT_CREATED_DATE_DESC); err != nil {
		log.Fatal(err)
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
	edit := true
	for edit {
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
		fmt.Fprint(&displayList,
			"Todo: "+task.Todo,
			comp,
			"\nPriority: "+task.Priority,
			"\nProjects + (space separated): "+strings.Join(task.Projects, " "),
			"\nContexts @ (space separated): "+strings.Join(task.Contexts, " "),
			"\nDue date yyyy-mm-dd: "+tdd,
			"\nThreshold yyyy-mm-dd: ",
			"\n",
			"\nDelete item",
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
		case strings.HasPrefix(out, "Complete item"):
			task.Completed = true
		case strings.HasPrefix(out, "Restore item"):
			task.Completed = false
		default:
			edit = false
		}
	}
	return *task
}

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
