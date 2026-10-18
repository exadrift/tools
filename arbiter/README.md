# arbiter
guided, gitops oriented tool for talos cluster management

## install
```
# binary will be installed to ~/.local/bin
# to install elsewhere, set BIN_DIR to the desired location
sh -c "curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/install.sh | sh -s -- arbiter"
```

## usage
arbiter is designed to work with single-node talos clusters, ideally suited to something like a home lab where the desire to have something up-and-running, reproducable, and easy-to-manage are the key considerations
- single node machine only
- store encrypted secrets in github, maintain a single configuration with keymaterial, for easy secure backup
- prepare a gitops friendly base for kubernetes configuration and application deployment

### prepare talos image
visit the talos image factory.  this typically assumes a bare-metal installation, however many other options can be used:
https://factory.talos.dev/
<br>the factory will let you configure an image for download, based on your hardware's needs
before proceeding to the project step, you should boot the target machine from the talos installer image.  this will get the target into "maintenance mode", which is necessary to receive fresh installation instructions

### initialize a project
first create a blank git repository and clone it locally.  this will store all your cluster configuration.  ideally, give it a name that will reflect your cluster name (recommended but not necessary)

```
# from the directory of the git repository root

arbiter init
```
this will walk you through the configuration process (machine ip, etc.).  at the end of this process basic configuration will be established at `~/.arbiter/<project-name>`.  you will be prompted to back up the `config.yaml` which will be generated there.  this file contains the configuration but most importantly the keymaterial which will be used to manage the secrets you store in the github repository.  if you lose this file, you won't be able to encrypt/decrypt those secrets anymore, which means you won't be able to contact the talos cluster.

### bootstrap talos
```
# if you're still in the git repo, or if not, just supply the project name as an argument

arbiter bootstrap
```
you will be prompted for the install disk on the target machine, then the process will begin.  the machine may reboot while this is underway, so make sure your install media is not the first to boot by default, this could cause installation to hang indefinitely.

### test kubectl
at the end of the bootstrap process, arbiter will configure kubectl.  you can now interrogate the new cluster via kubectl