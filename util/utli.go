package util

import (
	"os/exec"
)
func CmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}