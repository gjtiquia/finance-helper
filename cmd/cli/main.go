package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {

		case "--help", "-h":
			fmt.Println("TODO : help page")
			return

		case "connect":
			if len(os.Args) != 3 {
				fmt.Println("Usage: finance-helper connect <url>")
				return
			}

			if err := connect(os.Args[2]); err != nil {
				fmt.Println(err.Error())
				return
			}

			return

		case "status":
			if err := status(os.Stdout); err != nil {
				fmt.Println(err.Error())
			}
			return
		}
	}

	fmt.Println("TODO : help page")
}
