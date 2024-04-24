package scanner

import "time"

type file struct {
	Name     string
	Size     int
	Modified time.Time
}
