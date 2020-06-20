## Todotxtmenu

A dmenu/rofi script to view and manage
[todo.txt](https://github.com/todotxt/todo.txt-cli) lists.

### Installation

- `go get github.com/firecat53/todotxtmenu.go` OR [download binary](https://github.com/firecat53/todotxtmenu/releases)

### Usage

- Command line options:

          -archive=<true/false>
                Archive completed items to `done.txt` on exit (default true)
          -cmd string
                Dmenu command to use (dmenu, rofi, wofi, etc) (default "dmenu")
          -opts string
                Additional Rofi/Dmenu options (default "")
          -todo string
                Path to todo file (default "todo.txt")

- Configure Dmenu or Rofi using appropriate command line options or .Xresources
  and pass using the `-opts` flag to todotxtmenu.
  *NOTE* The `-i` flag is passed to both Dmenu and Rofi by default. The `-dmenu`
  flag is passed to Rofi. Examples:
  
        todotxtmenu -cmd rofi -todo /home/user/todo/todo.txt -opts "-theme todotxtmenu"
        todotxtmenu -todo /home/user/todo/todo.txt -opts
            "-fn SourceCodePro-Regular:12 -b -l 10 -nf blue -nb black"
