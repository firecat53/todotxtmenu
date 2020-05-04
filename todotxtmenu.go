package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

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
	var displayList strings.Builder
	todotxt.IgnoreComments = false

	tasklist, err := todotxt.LoadFromFilename("todo.txt")
	if err != nil {
		log.Fatal(err)
	}

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
	out := display(displayList, dmenuOpts)
	fmt.Println(out)
}

func display(list strings.Builder, opts []string) (result string) {
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd := exec.Command(opts[0], opts[1:]...)
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	cmd.Stdin = strings.NewReader(list.String())
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
