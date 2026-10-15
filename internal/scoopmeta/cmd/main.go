package main

import (
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/scoopmeta"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: scoopmeta INPUT OUTPUT")
		os.Exit(2)
	}
	if e := scoopmeta.Enrich(os.Args[1], os.Args[2]); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
