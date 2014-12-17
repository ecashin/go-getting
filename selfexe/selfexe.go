package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
)

const exe = "/tmp/selfexe-payload"

func main() {
	f, err := os.OpenFile(exe, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBufferString(Bin)
	// f.WriteString(Bin)
	// f.Close()
	// panic("early")
	g, err := gzip.NewReader(buf)
	if err != nil {
		panic(err)
	}
	io.Copy(f, g)
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
