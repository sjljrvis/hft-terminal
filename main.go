package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// root orchestrator: dispatches to the real binaries for convenience.
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		serverCmd(os.Args[2:])
	case "replay":
		runSubcommand("go", []string{"run", "./scripts/replay_ticks"})
	case "migrate":
		runSubcommand("go", []string{"run", "./scripts/migrate"})
	case "help", "-h", "--help":
		usage()
	default:
		log.Printf("unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func serverCmd(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	mode := fs.String("mode", "", "override mode: live|backtest")
	configPath := fs.String("config", "configs/dev.yaml", "path to YAML config")
	_ = fs.Parse(args)

	runArgs := []string{"run", "./server", "-config", *configPath}
	if *mode != "" {
		runArgs = append(runArgs, "-mode", *mode)
	}

	runSubcommand("go", runArgs)
}

func runSubcommand(name string, args []string) {
	log.Printf("running: %s %v", name, args)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("command failed: %v", err)
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  hft server [-config path] [-mode live|backtest]   # start API/webapp + executor")
	fmt.Println("  hft replay                         # run tick replay utility")
	fmt.Println("  hft migrate                        # run migrations")
}
