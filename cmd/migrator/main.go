package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kosovrzn/wb-tech-l0/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("PG_DSN environment variable is required")
	}

	ctx, cancel := migrate.ContextWithTimeout()
	defer cancel()

	var err error

	switch cmd {
	case "up":
		err = migrate.Up(ctx, dsn)
	case "down":
		steps, parseErr := migrate.ParseSteps(argOrEmpty(os.Args, 2))
		if parseErr != nil {
			log.Fatal(parseErr)
		}
		err = migrate.Down(ctx, dsn, steps)
	case "status":
		err = migrate.Status(ctx, dsn)
	case "redo":
		err = migrate.Redo(ctx, dsn)
	case "version":
		var v int64
		v, err = migrate.Version(ctx, dsn)
		if err == nil {
			fmt.Printf("current version: %d\n", v)
		}
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println("usage: migrator <up|down|status|redo|version> [steps]")
}

func argOrEmpty(args []string, idx int) string {
	if len(args) > idx {
		return args[idx]
	}
	return ""
}
