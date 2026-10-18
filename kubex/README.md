# kubex
a minimalistic kubernetes explorer TUI

- quickly switch between kubernetes contexts
- quickly switch between kubernetes namespaces
- execute commands in a shell to the side

## install
## install
```
# binary will be installed to ~/.local/bin
# to install elsewhere, set BIN_DIR to the desired location
sh -c "curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/install.sh | sh -s -- kubex"
```

## start TUI
```
kubex
```

## CLI help
```
kubex --help    # display the cli help
kubex --config  # modify the config file (keybindings)
```

## navigation / command
- `k` alias to execute `kubectl`
- `tab`/`shift+tab` switch pane right/left respectively
- `ctrl+pgup`/`ctrl+pgdn` scroll up/down in terminal view
