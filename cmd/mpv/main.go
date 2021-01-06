package main

import (
	"os"
	"strings"
	"syscall"
)

func main() {
	title := os.Args[2]
	msgLevel := os.Args[3]
	msgLevel = strings.ReplaceAll(msgLevel, ":", ",")
	restArgs := os.Args[4:]
	args := []string{"--title=" + title, msgLevel}
	args = append(args, restArgs...)
	syscall.Exec("/usr/local/bin/mpv", args, nil)
}
