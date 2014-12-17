package main

import (
	"os"
	"os/exec"
)

const exe = "/tmp/selfexe-payload"

func main() {
	f, err := os.OpenFile(exe, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	f.WriteString(Bin)
	f.Close()

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}
