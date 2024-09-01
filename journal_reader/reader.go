package journalreader

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type SshLogs struct {
	FailedRequest      int
	SuccessFullRequest int
}

var FailedRequestPattern = regexp.MustCompile(`Failed password for (\w+) from (\d+\.\d+\.\d+\.\d+)`)
var SuccessRequestPattern = regexp.MustCompile(`Accepted password for (\w+) from (\d+\.\d+\.\d+\.\d+)`)

const USERNAME_POSITION = 1
const HOST_POSITION = 2
const EXPECTED_NUMBER_OF_ELEMENTS_AFTER_SPLITTING = 3

// Count requests done to the OpenSSH Daemon documented by systemd
func CountRequest(journalLocation string, timeSpanInMinutes int) (SshLogs, map[string]SshLogs) {
	cmd := exec.Command("journalctl", "--file", journalLocation, "-u", "ssh", "--since", strconv.Itoa(timeSpanInMinutes)+" minutes ago")
	if timeSpanInMinutes == 0 {
		cmd = exec.Command("journalctl", "--file", journalLocation, "-u", "ssh")
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		panic("failed to execute journal command")
	}
	dataStr := string(output)
	lines := strings.Split(dataStr, "\n")

	logs := SshLogs{
		FailedRequest:      0,
		SuccessFullRequest: 0,
	}

	hosts := map[string]SshLogs{}

	for _, line := range lines {
		matches := SuccessRequestPattern.FindStringSubmatch(line)

		if len(matches) == EXPECTED_NUMBER_OF_ELEMENTS_AFTER_SPLITTING {
			host_ip := matches[HOST_POSITION]
			log, exist := hosts[host_ip]
			if !exist {
				log = SshLogs{
					FailedRequest:      0,
					SuccessFullRequest: 0,
				}
			}

			log.SuccessFullRequest++
			logs.SuccessFullRequest++
			hosts[host_ip] = log
			continue
		}

		matches = FailedRequestPattern.FindStringSubmatch(line)
		if len(matches) == EXPECTED_NUMBER_OF_ELEMENTS_AFTER_SPLITTING {

			host_ip := matches[HOST_POSITION]
			log, exist := hosts[host_ip]
			if !exist {
				log = SshLogs{
					FailedRequest:      0,
					SuccessFullRequest: 0,
				}
			}

			log.FailedRequest++
			logs.FailedRequest++
			hosts[host_ip] = log
			continue
		}
	}

	return logs, hosts
}
