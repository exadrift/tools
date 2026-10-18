# repo
git repository helper

- create a branch with all the remote tracking configs
- track a remote branch
- create a basic shell prompt with git branch display

## install
```
# binary will be installed to ~/.local/bin
# to install elsewhere, set BIN_DIR to the desired location
sh -c "curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/install.sh | sh -s -- repo"
```

## CLI help
```
repo branch create <name>       # create and push a branch and track remote
repo branch track <name>        # track a remote branch locally
repo prompt inject              # install a custom prompt with git branch display
```
