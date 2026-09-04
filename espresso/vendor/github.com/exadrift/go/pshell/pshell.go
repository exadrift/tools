package pshell

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type PShell struct {
	opt          Option
	lock         sync.Mutex
	sudoPassword string
}

type Option struct {
	PasswordCallback func() (string, error)
	Shell            string
}

type OptionFunc func(opt *Option)

func WithPasswordCallback(callback func() (string, error)) OptionFunc {
	return func(opt *Option) {
		opt.PasswordCallback = callback
	}
}

func WithShell(shellBin string) OptionFunc {
	return func(opt *Option) {
		opt.Shell = shellBin
	}
}

func New(optFuncs ...OptionFunc) *PShell {
	c := &PShell{
		opt: Option{},
	}

	for _, optFunc := range optFuncs {
		optFunc(&c.opt)
	}

	if c.opt.Shell == "" {
		shell := os.Getenv("SHELL")
		if shell == "" {
			c.opt.Shell = "/bin/bash"
		} else {
			c.opt.Shell = shell
		}
	}

	return c
}

func (c *PShell) GetSudoPassword() (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.sudoPassword != "" {
		return c.sudoPassword, nil
	}

	if c.opt.PasswordCallback == nil {
		return "", fmt.Errorf("no means of capturing sudo password was provided")
	}

	for {
		pass, err := c.opt.PasswordCallback()
		if err != nil {
			return "", err
		}

		cmd := exec.Command("sudo", "-S", "-k", "-v")
		cmd.Stdin = strings.NewReader(pass + "\n")
		if err := cmd.Run(); err == nil {
			return pass, nil
		}
	}
}

func (c *PShell) StoreSudoPassword(pw string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.sudoPassword = pw
}

func (c *PShell) execute(cmd *exec.Cmd, opts ...ExecOptionFunc) (string, error) {
	var execOption ExecOption
	for _, opt := range opts {
		opt(&execOption)
	}

	if len(execOption.EnvVars) > 0 {
		cmd.Env = append(os.Environ(), execOption.EnvVars...)
	}

	var capture bytes.Buffer
	var wg sync.WaitGroup

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("unable to start command in a pty: %w", err)
	}
	defer func() {
		_ = ptyFile.Close()
	}()

	wg.Add(1)
	cancelChan := make(chan struct{}, 1)
	go func() {
		defer wg.Done()
		timer := time.NewTimer(time.Second)
		for {
			select {
			case <-cancelChan:
				timer.Stop()
				return
			case <-timer.C:
				func() {
					defer timer.Reset(time.Second)
					isSudo, sudoPid, err := isRunningSudo(int32(cmd.Process.Pid))
					if err != nil {
						return
					}
					if isSudo {
						// wait to ensure being prompted
						time.Sleep(time.Second)
						secondIsSudo, secondSudoPid, err := isRunningSudo(int32(cmd.Process.Pid))
						if err != nil {
							return
						}

						if !secondIsSudo || secondSudoPid != sudoPid {
							isSudo = false
						}
					}

					if isSudo {
						sudoPassword, err := c.GetSudoPassword()
						if err != nil {
							return
						}
						c.StoreSudoPassword(sudoPassword)

						_, err = fmt.Fprintf(ptyFile, "%s\n", sudoPassword)
						if err != nil {
							return
						}
					}
				}()
			}
		}

	}()

	buf := make([]byte, 4096)
	for {
		n, err := ptyFile.Read(buf)
		if err != nil {
			if errors.Is(err, syscall.EIO) || errors.Is(err, io.EOF) {
				break
			}

			return "", fmt.Errorf("error reading pty: %w", err)
		}
		if n > 0 {
			_, err := capture.Write(buf[:n])
			if err != nil {
				return "", err
			}
		}
	}

	if err = cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("exited with error code %d\n%s", exitErr.ExitCode(), capture.String())
		}
		return "", err
	}

	close(cancelChan)
	wg.Wait()

	return capture.String(), nil
}

type ExecOption struct {
	EnvVars []string
}

type ExecOptionFunc func(*ExecOption)

// WithEnvVars takes a mapping of key/value pairs
func WithEnvVars(envVars map[string]string) ExecOptionFunc {
	return func(eo *ExecOption) {
		eo.EnvVars = make([]string, len(envVars))
		i := 0
		for k, v := range envVars {
			eo.EnvVars[i] = fmt.Sprintf("%s=%s", k, v)
			i++
		}

	}
}

func (c *PShell) Execute(filename string, opts ...ExecOptionFunc) (string, error) {
	cmd := exec.Command(c.opt.Shell, filename)
	return c.execute(cmd, opts...)
}

func (c *PShell) ExecuteCommands(commands string, opts ...ExecOptionFunc) (string, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = reader.Close()
	}()

	_, err = writer.WriteString(commands)
	if err != nil {
		return "", fmt.Errorf("unable write commands to pipe: %w", err)
	}
	_ = writer.Close()

	filename := "/dev/fd/3"
	cmd := exec.Command(c.opt.Shell, "-c", fmt.Sprintf("source %s", filename))
	cmd.ExtraFiles = []*os.File{reader}

	result, err := c.execute(cmd, opts...)
	if err != nil {
		return "", err
	}

	return result, err
}
