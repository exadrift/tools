package crucible

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/exadrift/go/kv"
	"github.com/exadrift/tools/crucible/internal/ssh"
	"github.com/goccy/go-yaml"
	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	SshdImage              string = "crucible/test-containers/sshd:latest"
	SshAgentImage          string = "crucible/test-containers/ssh-agent:latest"
	AgentUnixSocketDir     string = "/tmp/sshtest"
	ContainerUnixSocketDir string = "/etc/sshtest"
	SshPort                string = "22"
	TestPassphrase         string = "testphrase"
)

var (
	CompletionFile string = filepath.Join(ContainerUnixSocketDir, "complete")
)

type CrucibleTestSuite struct {
	suite.Suite

	sshContainer      testcontainers.Container
	sshAgentContainer testcontainers.Container
	sshHost           string
	testDataDir       string
}

func (suite *CrucibleTestSuite) SetupTest() {
	testDataDir, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	suite.testDataDir = testDataDir
	suite.NoError(err)

	configScript := filepath.Join(testDataDir, "config.sh")
	pubKey := filepath.Join(testDataDir, "id_ed25519.pub")
	pubKeyPhrase := filepath.Join(testDataDir, "id_ed25519_passphrase.pub")
	suite.NoError(err)

	req := testcontainers.ContainerRequest{
		Image:           SshdImage,
		AlwaysPullImage: false,
		WaitingFor:      wait.ForFile("/home/test/done.file").WithStartupTimeout(10 * time.Second),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      configScript,
				ContainerFilePath: "/etc/config/config.sh",
				FileMode:          0o555,
			},
			{
				HostFilePath:      pubKey,
				ContainerFilePath: "/tmp/id_ed25519.pub",
			},
			{
				HostFilePath:      pubKeyPhrase,
				ContainerFilePath: "/tmp/id_ed25519_passphrase.pub",
			},
		},
		ExposedPorts: []string{SshPort},
	}
	suite.sshContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.NoError(err)

	sshHost, err := suite.sshContainer.Host(context.Background())
	suite.NoError(err)

	sshPort, err := suite.sshContainer.MappedPort(context.Background(), SshPort)
	suite.NoError(err)

	suite.sshHost = fmt.Sprintf("%s:%s", sshHost, sshPort.Port())

	_ = os.RemoveAll(AgentUnixSocketDir)
	err = os.MkdirAll(AgentUnixSocketDir, 0777)
	suite.NoError(err)

	cUser, err := user.Current()
	suite.NoError(err)

	socketFile := filepath.Join(ContainerUnixSocketDir, "agent.sock")
	req = testcontainers.ContainerRequest{
		Image:           SshAgentImage,
		AlwaysPullImage: false,
		WaitingFor:      wait.ForFile(CompletionFile).WithStartupTimeout(10 * time.Second),
		Env: map[string]string{
			"COMPLETION_FILE": CompletionFile,
			"SSH_AUTH_SOCK":   socketFile,
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = []string{
				fmt.Sprintf("%s:%s", AgentUnixSocketDir, ContainerUnixSocketDir),
			}
		},
		ConfigModifier: func(c *container.Config) {
			c.User = fmt.Sprintf("%s:%s", cUser.Uid, cUser.Gid)
		},
	}

	suite.sshAgentContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.NoError(err)

	err = ssh.InitAgentInstance(ssh.WithSshAuthSock(filepath.Join(AgentUnixSocketDir, "agent.sock")))
	suite.NoError(err)
}

func (suite *CrucibleTestSuite) TearDownTest() {
	if suite.sshContainer != nil {
		testcontainers.CleanupContainer(suite.T(), suite.sshContainer)
	}

	if suite.sshAgentContainer != nil {
		testcontainers.CleanupContainer(suite.T(), suite.sshAgentContainer)
	}

	_ = os.RemoveAll(AgentUnixSocketDir)
}

func (suite *CrucibleTestSuite) TestEndToEnd() {
	location := "/mnt/end-to-end"
	_, err := os.Stat(location)
	if err != nil {
		// if running in the debugger, we need to make sure we use the appropriate path
		location, err = filepath.Abs(filepath.Join("..", "..", "mnt", "end-to-end"))
		suite.NoError(err)
	} else {
		// run the initialization script for the runner
		cmd := exec.Command("sh", "-c", "./init.sh")
		cmd.Dir = "/mnt"
		b, err := cmd.Output()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				suite.T().Log(string(exitErr.Stderr))
			} else {
				suite.T().Log(err)
			}
		}
		suite.NoError(err)
		suite.T().Log(string(b))
	}

	extraConfig := kv.NewStore()
	err = extraConfig.Set(suite.sshHost, "hosts", "testServer", "host")
	suite.NoError(err)
	err = extraConfig.Set("test", "hosts", "testServer", "ssh", "user")
	suite.NoError(err)
	err = extraConfig.Set("/root/.ssh/id_ed25519", "hosts", "testServer", "ssh", "keyPath")
	suite.NoError(err)
	extraConfigBytes, err := yaml.Marshal(extraConfig.GetMapping())
	suite.NoError(err)

	extraConfigFile := "/tmp/extraHostConfig.yaml"
	err = os.WriteFile(extraConfigFile, extraConfigBytes, 0666)
	suite.NoError(err)

	jsonResult, err := ExecuteSequenceFromCwd(
		location,
		[]string{filepath.Join(location, "config.yaml"), extraConfigFile},
		nil,
		"test",
		[]string{"testServer"},
		true,
		true,
	)
	suite.NoError(err)
	m := map[string]any{}
	err = json.Unmarshal(jsonResult, &m)
	suite.NoError(err)
	suite.Zero(m["failCount"].(float64))

	suite.T().Log("logging the unmarshal result")
	suite.T().Log(string(jsonResult))
}

func TestCrucible(t *testing.T) {
	suite.Run(t, new(CrucibleTestSuite))
}
