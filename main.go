package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	// Create Server and Route Handlers
	fmt.Println(8877)
	cmd := exec.Command("/usr/local/bin/janus")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	go func() {

		fmt.Printf("Waiting for Janus to finish...\n")
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Janus finished with error: %v\n", err)
			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0
				// This works on both Unix and Windows. Although package
				// syscall is generally platform dependent, WaitStatus is
				// defined for both Unix and Windows and in both cases has
				// an ExitStatus() method with the same signature.
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					fmt.Printf("Janus Exit Status: %d\n", status.ExitStatus())
					os.Exit(status.ExitStatus())
				}
			} else {
				panic("fail deref err")
			}
		} else {
			fmt.Printf("Janus finished without errors\n")
		}
		os.Exit(-1)
	}()

	select {}
}
