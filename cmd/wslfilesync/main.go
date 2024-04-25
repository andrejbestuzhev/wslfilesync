package main

import (
	"flag"
	"wslfilesync/m/v2/internal/scanner"
)

func main() {

	primary := flag.String("a", "", "Main dir")
	secondary := flag.String("b", "", "Sync with")
	flag.Parse()
	s := scanner.NewScanner(*primary, *secondary)
	s.Run()
}
