// servez is the ServeEz CLI: init (control plane), status, deploy, scale,
// token, and (agent side) join. Commands register via the Command registry
// so each command lives in its own file and parallel agents don't collide.
package main

import (
	"fmt"
	"os"
)

// Command is a single CLI subcommand.
type Command struct {
	// Name is the subcommand word, e.g. "init".
	Name string
	// Usage is a short usage line.
	Usage string
	// Run executes the command with args after the subcommand word.
	Run func(args []string) error
}

var commands []Command

// registerCommand adds a command to the global registry (called from init()).
func registerCommand(c Command) {
	commands = append(commands, c)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	name := os.Args[1]
	for _, c := range commands {
		if c.Name == name {
			if err := c.Run(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "servez %s: %v\n", name, err)
				os.Exit(1)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "unknown command %q\n", name)
	usage()
	os.Exit(2)
}

func usage() {
	fmt.Println("usage: servez <command> [args]")
	for _, c := range commands {
		fmt.Printf("  %s\n", c.Usage)
	}
}
