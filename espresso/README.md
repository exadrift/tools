# espresso
a decentralized package management tool

## what do you mean decentralized?
in a centralized package management tool, packages are stored in a centralized repository.  espresso is designed to manage packages from any source, focusing on installation and version tracking particulars

## usage
start by installing espresso, you may need to run the command as `sudo` in order to place the contents in /usr/local/bin/
```
sh -c "curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/install.sh | sh -s -- espresso"
```

next, define a package manifest, this tells espresso what to keep in sync.  here's an example manifest with a single item - we'll go over the details below

```
packages:
  - name: helm
    version: v4.0.0
    getVersion: helm version --template {{.Version}}
    sync: |
      curl -fsSL -o /tmp/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-4
      chmod 700 /tmp/get_helm.sh
      DESIRED_VERSION=${VERSION} /tmp/get_helm.sh
      rm /tmp/get_helm.sh
```

items in the `packages` list will be synced in order of occurrence, thus dependencies should be listed before consumers, in order to ensure they are available first.

- `name` superficially describes the name of the package being installed, it's for labelling purpopses only
- `version` describes the target version to be installed.  running sync will ensure that version is the version on system
- `getVersion` indicates the command which should be run on the host system, to obtain the version.  this command needs to output a string that exactly matches the version in `version` in order to properly sync
- `sync` describes the command set to run, in order to install a particular version of the package.  this will be executed as the user running espresso.  any `sudo` calls will be forwarded to the executor.  this means espresso can safely run as a non-privileged user, where user specifics are required as part of the installation, in addition to being able to execute privileged instructions when necessary.  `sudo` priviledge once intercepted is cached for the remainder of the espresso sync session, meaning you will only be prompted once per session if running as a non-privileged user.  the VERSION environment variable will be set to match the version supplied in the `version` attribute of the package definition.
- `syncScript` accepts a path to a script which will perform the sync - the path can be relative to the location of the manifest file, or absolute.  the VERSION environment variable will be set to match the version supplied in the `version` attribute of the package definition.  this is typically used for larger scripts which would be too large to sit inline

running `espresso`
```
# display cli help
espresso --help

# run a full sync (installation/verification of packages)
espresso sync <manifest-path>

# use the --verbose option for detailed command output and error diagnosis
```