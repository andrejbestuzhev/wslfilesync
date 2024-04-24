package main

import (
	"flag"
	"wslfilesync/m/v2/internal/scanner"
)

func main() {

	primary := flag.String("primary", "", "Main dir")
	secondary := flag.String("secondary", "", "Sync with")
	flag.Parse()
	s := scanner.NewScanner(*primary, *secondary)
	s.Run()
}
