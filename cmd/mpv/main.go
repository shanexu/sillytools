package main

import (
	"os"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) >= 6 {
		title := os.Args[2]
		msgLevel := os.Args[3]
		msgLevel = strings.ReplaceAll(msgLevel, ":", ",")
		restArgs := os.Args[4:]
		args := []string{"--vid=no", "--title=" + title, msgLevel}
		args = append(args, restArgs...)
		syscall.Exec("/usr/local/bin/mpv", args, nil)
	} else {
		syscall.Exec("/usr/local/bin/mpv", os.Args[1:], nil)
	}
}
