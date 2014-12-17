package main

import (
	"fmt"
	"os"
	"os/exec"
)

const exe = "/tmp/selfexe-payload"

func main() {
	fmt.Println("hi", len(Bin))
	f, err := os.OpenFile(exe, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	f.WriteString(Bin)
	f.Close()

	cmd := exec.Command(exe)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	cmd.Wait()
}
