package pshell

import "github.com/shirou/gopsutil/v4/process"

func isRunningSudo(pid int32) (bool, int32, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false, 0, err
	}

	name, err := p.Name()
	if err != nil {
		return false, 0, err
	}

	if name == "sudo" {
		return true, p.Pid, nil
	}

	children, err := p.Children()
	if err != nil {
		return false, 0, err
	}

	for _, child := range children {
		isRunningSudo, sudoPid, err := isRunningSudo(child.Pid)
		if err != nil {
			continue
		}

		if isRunningSudo {
			return true, sudoPid, nil
		}
	}

	return false, 0, nil
}
