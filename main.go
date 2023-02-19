package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// for colored output
const reset = "\033[0m"
const red = "\033[31m"
const green = "\033[32m"
const cyan = "\033[36m"
const bold = "\033[1m"

// command line arguments
var hostFile string
var hostString string
var hosts []string
var command string
var user string
var sshOptions string

// Waiting group for goroutines
var wg = sync.WaitGroup{}

func formatOutput(id int, host string, err error, stdout []byte, stderr []byte) {
	// Example output:
	// [1] 15:04:00 [SUCCESS] host.hostname.com
	// [0] 21:18:57 [FAILURE] host.hostname.com exit status 127
	var message string
	// Format output message depends on command status
	if err != nil {
		message = fmt.Sprintf("%s[%sFAILURE]%s %s %s%s%s", red, bold, reset, host, red, err, reset)
	} else {
		message = fmt.Sprintf("%s[%sSUCCESS]%s %s", green, bold, reset, host)
	}

	// Append stdout to output
	if len(stdout) != 0 {
		formattedStdout := fmt.Sprintf("\n%vStdout:%v %s", green, reset, stdout)
		message = fmt.Sprint(message, formattedStdout)
	}

	// Append stderr to output
	if len(stderr) != 0 {
		formattedStderr := fmt.Sprintf("\n%vStderr:%v %s", red, reset, stderr)
		message = fmt.Sprint(message, formattedStderr)
	}

	// Add colors to thread ID
	coloredId := fmt.Sprintf("%s[%d]%s", cyan, id, reset)
	currentTime := time.Now().Format("15:04:05")
	// Print everything in a single message
	fmt.Println(coloredId, currentTime, message)
}

func runSSHCommand(id int, host string, sshArgs []string) {
	// Add hostname to command args
	sshArgs = append([]string{host}, sshArgs...)
	// Create command, but not run
	cmd := exec.Command("ssh", sshArgs...)

	// Connect to command stderr
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	// Connect to command stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Get bytes arrays output from stdout and stderr
	stdout, _ := io.ReadAll(stdoutPipe)
	stderr, _ := io.ReadAll(stderrPipe)

	// Waiting until command is finished
	err = cmd.Wait()

	// Format and print output
	formatOutput(id, host, err, stdout, stderr)
	wg.Done()
}

// Get, prase and validate command line arguments
func parseCommandLineArguments() {
	flag.StringVar(&hostFile, "h", "", "file with list of hosts")
	flag.StringVar(&hostString, "H", "", "list of hosts separated by spaces")
	flag.StringVar(&command, "i", "", "command to execute")
	flag.StringVar(&user, "l", "", "specifies the user to log in as on the remote machine")
	flag.StringVar(&sshOptions, "o", "", "additional ssh option in quotes and separated by spaces")
	flag.StringVar(&command, "u", "", "remote user")
	flag.Parse()

	// This parameters is required
	if len(hostFile) == 0 && len(hostString) == 0 {
		fmt.Println("Please specify -h or -H option")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// This parameter is required
	if len(command) == 0 {
		fmt.Println("Please specify command to execute")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

// Parse and validate hosts string
func parseHostsString(hostsString string) []string {
	hosts = strings.Fields(hostString)

	if len(hosts) == 0 {
		log.Fatal("can't parse hosts string")
		os.Exit(1)
	}
	return hosts
}

// Parse and validate list of hosts from host file
func parseHostFile(hostFile string) []string {
	file, err := os.Open(hostFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hosts = append(hosts, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if len(hosts) == 0 {
		log.Fatal("hosts file in empty")
	}
	return hosts
}

func combineSSHCommand() []string {
	sshArgs := []string{}

	if len(user) != 0 {
		sshArgs = append(sshArgs, "-l")
		sshArgs = append(sshArgs, user)
	}

	if len(sshOptions) != 0 {
		parsedSSHOptions := strings.Fields(sshOptions)
		for _, option := range parsedSSHOptions {
			sshArgs = append(sshArgs, "-o")
			sshArgs = append(sshArgs, option)
		}
	}

	sshArgs = append(sshArgs, command)
	return sshArgs
}

func main() {
	parseCommandLineArguments()
	if len(hostString) == 0 {
		hosts = parseHostFile(hostFile)
	} else {
		hosts = parseHostsString(hostString)
	}
	sshArgs := combineSSHCommand()
	for i, host := range hosts {
		wg.Add(1)
		go runSSHCommand(i, host, sshArgs)
	}
	wg.Wait()
}
