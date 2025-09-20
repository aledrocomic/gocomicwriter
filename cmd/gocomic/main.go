package main

import (
	"fmt"
	"os"

	"gocomic/internal/version"
)

func main() {
	// Minimal CLI entrypoint for the Go Comic Writer project.
	// For now, it prints a banner and an optional version.
	args := os.Args
	if len(args) > 1 {
		switch args[1] {
		case "version", "--version", "-v":
			fmt.Println(version.String())
			return
		}
	}

	fmt.Println("Go Comic Writer — development skeleton")
	fmt.Printf("Version: %s\n", version.String())
}
