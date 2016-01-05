package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

const (
	DefaultMonitorInterval = 10 * time.Second
	DefaultStartWait       = 10 * time.Second

	ProcessStatIndex = 2

	ProcessStatRunning         = "R"
	ProcessStatSleeping        = "S"
	ProcessStatStoped          = "T"
	ProcessStatZombie          = "Z"
	ProcessStatUninterruptible = "D"
)

// monitor a process
func monitorProcess(ps *Process, complete chan int) {
	for {

		// monitor process
		logger.Println("monitor process " + ps.Name + " by pid file " + ps.PidFile)

		pid, err := getPidFromFile(ps.PidFile)
		if err != nil {
			logger.Println("failed to get pid, error: " + err.Error())
			startProcess(ps)
			continue
		}

		running, err := isProcessRunning(pid)
		if err != nil {
			logger.Println("failed to get process stat, error: " + err.Error())
			startProcess(ps)
			continue
		}

		if !running {
			logger.Println("process with pid " + pid + " not running")
			startProcess(ps)
			continue
		}

		logger.Println("process with pid " + pid + " is running")

		// sleep a while
		d, intervalError := time.ParseDuration(ps.Interval)
		if intervalError != nil {
			d = DefaultMonitorInterval
		}
		time.Sleep(d)

	}
	complete <- 1
}

// get pid from pid file
func getPidFromFile(file string) (string, error) {
	pidBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	pid := strings.TrimSpace(string(pidBytes))
	if pid == "" {
		return "", fmt.Errorf("pid file is empty")
	}
	return pid, nil
}

// check process running or not by pid
func isProcessRunning(pid string) (bool, error) {
	stat, err := getProcessStatByPid(pid)
	if err != nil {
		return false, err
	}

	if stat == ProcessStatStoped || stat == ProcessStatZombie {
		return false, nil
	}
	return true, nil
}

// read /proc/$pid/stat to get process stat
func getProcessStatByPid(pid string) (string, error) {
	file := "/proc/" + pid + "/stat"
	statBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	stat := strings.TrimSpace(string(statBytes))
	statList := strings.Split(stat, " ")

	if len(statList) < (ProcessStatIndex + 1) {
		return "", fmt.Errorf("stat file is empty")
	}

	return strings.TrimSpace(statList[ProcessStatIndex]), nil
}

// try to start process
func startProcess(ps *Process) {
	shell := fmt.Sprintf("nohup %s 1%s 2%s &", ps.Command, ps.StdOut, ps.StdErr)

	logger.Println(fmt.Sprintf("try to start %s, exec bash command: %s", ps.Name, shell))

	_, err := runShell(shell, ps.User)
	if err != nil {
		logger.Println(fmt.Sprintf("start %s failed, error: %v", ps.Name, err))
		return
	}

	// wait process start
	d, err := time.ParseDuration(ps.StartWait)
	if err != nil {
		d = DefaultStartWait
	}
	time.Sleep(d)

}

func runShell(shell string, userName string) ([]byte, error) {
	uc, _ := user.Current()
	if userName != "" && userName != uc.Name {
		if uc.Name == "root" {
			shell = fmt.Sprintf(
				"/sbin/runuser %s -c \"%s\"",
				userName,
				strings.Replace(shell, "\"", "\\\"", -1),
			)
		} else {
			shell = fmt.Sprintf(
				"sudo -u %s %s",
				userName,
				shell,
			)
		}
	}

	cmd := exec.Command("/bin/bash", "-c", shell)
	return cmd.CombinedOutput()
}
