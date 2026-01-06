package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"obsite/internal/generator"
	"obsite/internal/server"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	serve := flag.Bool("serve", false, "Start development server with live reload")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-serve] <source> [target]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  source  Source directory containing markdown files\n")
		fmt.Fprintf(os.Stderr, "  target  Target directory for generated site (default: site/)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 || len(args) > 2 {
		flag.Usage()
		os.Exit(1)
	}

	source := args[0]
	target := "site/"
	if len(args) == 2 {
		target = args[1]
	} else if !*serve {
		// Non-serve mode requires both args for backward compatibility
		flag.Usage()
		os.Exit(1)
	}

	// Validate source exists
	if _, err := os.Stat(source); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: source directory does not exist: %s\n", source)
		os.Exit(1)
	}

	gen := generator.New(source, target, templateFS)

	// Include drafts when running in serve mode
	if *serve {
		gen.IncludeDrafts = true
	}

	if *serve {
		srv := server.New(gen, source, target)
		if err := srv.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := gen.Generate(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
