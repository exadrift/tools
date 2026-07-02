package ssh

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

type SshInfo struct {
	User           string
	KeyPath        string
	KnownHostsPath string
	Hostname       string
	Port           int
}

func last(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	l := len(vals) - 1
	return vals[l]
}

func GetSshInfo(hostAlias string) (*SshInfo, error) {
	sshInfo := &SshInfo{}

	sshInfo.User = last(ssh_config.GetAll(hostAlias, "User"))
	sshInfo.KeyPath = last(ssh_config.GetAll(hostAlias, "IdentityFile"))
	knownHostsFile := last(ssh_config.GetAll(hostAlias, "UserKnownHostsFile"))
	sshInfo.Hostname = last(ssh_config.GetAll(hostAlias, "Hostname"))

	// can contain multiple paths here, separated by spaces
	if strings.Contains(knownHostsFile, " ") {
		knownHostsFile = strings.Split(knownHostsFile, " ")[0]
	}
	sshInfo.KnownHostsPath = knownHostsFile

	if sshInfo.User == "" {
		curUser, err := user.Current()
		if err != nil {
			return nil, err
		}
		sshInfo.User = curUser.Username
	}

	var hostname string
	var port = "22"
	parts := strings.Split(hostAlias, ":")
	if len(parts) == 1 {
		hostname = parts[0]
	} else if len(parts) == 2 {
		hostname = parts[0]
		port = parts[1]
	}

	i, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse port from ssh configuration for %s: %w", hostAlias, err)
	}
	sshInfo.Port = int(i)

	if sshInfo.Hostname == "" {
		sshInfo.Hostname = hostname
	}

	if strings.Contains(sshInfo.KeyPath, "~") || strings.Contains(sshInfo.KnownHostsPath, "~") {
		u, err := user.Lookup(sshInfo.User)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve user \"%s\" when getting ssh info for %s: %w", sshInfo.User, hostAlias, err)
		}

		sshInfo.KeyPath = strings.ReplaceAll(sshInfo.KeyPath, "~", u.HomeDir)
		sshInfo.KnownHostsPath = strings.ReplaceAll(sshInfo.KnownHostsPath, "~", u.HomeDir)
	}

	// when a host provided by the user contains a port number, allow this port number to supercede the one
	// provided by the ssh config
	if strings.Contains(sshInfo.Hostname, ":") {
		parts := strings.SplitN(sshInfo.Hostname, ":", 2)
		sshInfo.Hostname = parts[0]
		i, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse port from ssh configuration for %s: %w", hostAlias, err)
		}
		sshInfo.Port = int(i)
	}

	return sshInfo, nil
}
