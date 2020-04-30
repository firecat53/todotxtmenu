## Todotxtmenu

A dmenu/rofi script to view and manage
[todo.txt](https://github.com/todotxt/todo.txt-cli) lists.

- Copy or symlink the script to your bin folder. `todo.sh` should be in your
  $PATH
- Create a keybinding to activate the script
- Edit the configuration file if desired to add dmenu options or use a
  dmenu replacement such as Rofi:

```ini
[dmenu]
# dmenu_command = /usr/bin/dmenu
# dmenu_command = /usr/bin/rofi -width 30 -theme todo
# Rofi and dmenu are set to case insensitive by default `-i`
# fn = -*-terminus-medium-*-*-*-16-*-*-*-*-*-*-*
# fn = font string
# nb = normal background (name, #RGB, or #RRGGBB)
# nf = normal foreground
# sb = selected background
# sf = selected foreground
# b =  (just set to empty value and menu will appear at the bottom
# m = number of monitor to display on
```
