package scanner

import (
	"io/ioutil"
	"log"
)

type Directory struct {
	path        string
	size        uint64
	files       uint64
	directories uint64
}

func (d *Directory) Info() {
	files, size, directories, err := getStats(d.path)
	if err != nil {
		log.Panicln(err)
	}
	d.files = files
	d.size = size
	d.directories = directories
}

func (d *Directory) Equals(d2 *Directory) bool {
	return d.directories == d2.directories && d.files == d2.files && d.size == d2.size
}

func getStats(dirPath string) (uint64, uint64, uint64, error) {
	var totalSize uint64
	var totalFiles uint64
	var totalSubdirs uint64

	var getDirStat func(path string) error
	getDirStat = func(path string) error {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() {
				totalSubdirs++
			} else {
				totalFiles++
				totalSize += uint64(file.Size())
			}
		}
		return nil
	}

	if err := getDirStat(dirPath); err != nil {
		return 0, 0, 0, err
	}

	return totalFiles, totalSize, totalSubdirs, nil
}
