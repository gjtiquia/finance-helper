package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {

		case "--help", "-h":
			printHelp(os.Stdout)
			return

		case "connect":
			if len(os.Args) != 3 {
				fmt.Fprintln(os.Stdout, "Usage: finance-helper connect <url>")
				return
			}

			if err := connect(os.Stdout, os.Args[2]); err != nil {
				fmt.Fprintln(os.Stdout, err.Error())
				return
			}

			return

		case "status":
			if err := status(os.Stdout); err != nil {
				fmt.Fprintln(os.Stdout, err.Error())
			}
			return
		}
	}

	printHelp(os.Stdout)
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "finance-helper")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  finance-helper <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  connect <url>  Save and verify the server URL")
	fmt.Fprintln(w, "  status         Show config and server status")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  -h, --help     Show help")
}
