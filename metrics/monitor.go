package metrics

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	monitorPeriod = 5 * time.Second
)

var (
	monitorMet = New("monitor")
	// the shell command doesn't work on Mac so provide a flag to turn if off on Mac
	disableFDMonitor = flag.Bool("disable_fd_monitor", true, "Disable FD monitor")
)

// StartMonitor start the monitor
func StartMonitor() {
	if *disableFDMonitor {
		return
	}
	go func() {
		for range time.Tick(monitorPeriod) {
			monitorFdCount()
			if runtime.GOOS != "darwin" {
				monitorFileNr()
			}
		}
	}()
}

func monitorFdCount() {
	var ulimitN int
	var fdCount int

	pid := os.Getpid()
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)

	ulimitNCmd := "ulimit -n"
	ulimitNStr, cmdErr := runShCommand(ulimitNCmd)
	if cmdErr != nil {
		return
	}
	fmt.Sscanf(ulimitNStr, "%d", &ulimitN)

	countFdCmd := fmt.Sprintf("ls %s | wc -l", fdDir)
	fdCountStr, cmdErr := runShCommand(countFdCmd)
	if cmdErr != nil {
		return
	}
	fmt.Sscanf(fdCountStr, "%d", &fdCount)

	defer monitorMet.BumpHistogram("monitorFdCount.count", float64(fdCount), "monitor", "fd")
	defer monitorMet.BumpHistogram("monitorFdCount.ulimit", float64(ulimitN), "monitor", "fd")
}

func monitorFileNr() {
	readFileNrCmd := "cat /proc/sys/fs/file-nr"
	fileNrStr, cmdErr := runShCommand(readFileNrCmd)
	if cmdErr != nil {
		return
	}
	var allocatedFileHandle, zero, maxFileHandle int
	fmt.Sscanf(fileNrStr, "%d %d %d", &allocatedFileHandle, &zero, &maxFileHandle)

	defer monitorMet.BumpHistogram("monitorFileNr.allocated", float64(allocatedFileHandle), "monitor", "filehandle")
	defer monitorMet.BumpHistogram("monitorFileNr.max", float64(maxFileHandle), "monitor", "filehandle")
}

func runShCommand(cmd string) (output string, cmdErr error) {
	args := []string{"-c", cmd}
	outputBytes, cmdErr := exec.Command("/bin/sh", args...).Output()
	if cmdErr != nil {
		log.WithField("cmdErr", cmdErr).Error("runShCommand failed")
		return "", cmdErr
	}
	return string(outputBytes), nil
}
