# kubex
a minimalistic kubernetes explorer TUI

- quickly switch between kubernetes contexts
- quickly switch between kubernetes namespaces
- execute commands in a shell to the side

## install
```
# run as sudo if you don't have root privileges in order to drop the binary in /usr/local/bin
sh -c "curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/install.sh | sh -s -- kubex v0"
```

## start TUI
```
kubex
```

## navigation / command
- `k` alias to execute `kubectl`
- `tab`/`shift+tab` switch pane right/left respectively
- `ctrl+p` execute a command which has been fully typed but `<enter>` has not yet been pressed.  this will execute the command and pipe the output to `vi` for viewing with scrollability