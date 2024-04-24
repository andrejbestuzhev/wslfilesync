package scanner

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

type Directory struct {
	path         string
	size         uint64
	tfiles       uint64
	tdirectories uint64
	files        []file
	lastModified time.Time
}

func (d *Directory) Info() {

	files, tfiles, size, directories, lastModified, err := getStats(d.path)
	if err != nil {
		log.Panicln(err)
	}
	d.tfiles = tfiles
	d.size = size
	d.tdirectories = directories
	d.files = files
	d.lastModified = lastModified
}

func strDiff(a []string, b []string) []string {
	array2Map := make(map[string]bool)
	for _, str := range b {
		array2Map[str] = true
	}

	var diff []string
	for _, str := range a {
		if !array2Map[str] {
			diff = append(diff, str)
		}
	}

	return diff
}

func prepareStringsFromFiles(aDir Directory, bDir Directory) ([]string, []string) {
	var aDirFiles []string
	var bDirFiles []string

	for _, f := range aDir.files {
		aDirFiles = append(aDirFiles, f.Name)
	}

	for _, f := range bDir.files {
		bDirFiles = append(bDirFiles, f.Name)
	}
	return aDirFiles, bDirFiles
}

func (d *Directory) GetUpdateFiles(bDir *Directory) []string {
	var res []string
	for i, file := range d.files {
		if d.files[i].Size != bDir.files[i].Size {
			res = append(res, fmt.Sprintf("%s/%s", d.path, file.Name))
		}
	}
	return res
}

func (d *Directory) GetNewFiles(bDir *Directory) []string {
	a, b := prepareStringsFromFiles(*d, *bDir)
	diff := strDiff(a, b)
	var res []string
	for _, f := range diff {
		res = append(res, fmt.Sprintf("%s/%s", d.path, f))
	}
	return res
}

func (d *Directory) GetRemovedFiles(bDir *Directory) []string {

	a, b := prepareStringsFromFiles(*bDir, *d)
	diff := strDiff(a, b)
	var res []string
	for _, f := range diff {
		res = append(res, fmt.Sprintf("%s/%s", d.path, f))
	}
	return res
}

func (d *Directory) GetNewDirectories() {

}

func (d *Directory) GetRemovedDirectories() {

}

func (d *Directory) GetFileIndexByString(filename string) (int, bool) {
	for i, f := range d.files {
		if f.Name == filename {
			return i, true
		}
	}
	return 0, false
}

func (d *Directory) Equals(d2 *Directory) bool {
	return d.tdirectories == d2.tdirectories &&
		d.tfiles == d2.tfiles &&
		d.size == d2.size &&
		d.lastModified.Equal(d2.lastModified)
}

func getStats(dirPath string) ([]file, uint64, uint64, uint64, time.Time, error) {
	var totalSize uint64
	var totalFiles uint64
	var totalSubdirs uint64
	var files []file
	var lastModified time.Time

	var getDirStat func(path string) error
	getDirStat = func(path string) error {
		filesInDir, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		for _, f := range filesInDir {
			if f.IsDir() {
				totalSubdirs++
			} else {
				totalFiles++
				files = append(files, file{
					Name:     f.Name(),
					Size:     int(f.Size()),
					Modified: f.ModTime(),
				})
				totalSize += uint64(f.Size())
			}
		}

		var maxDate = time.Time{}
		for _, f := range files {
			if f.Modified.After(maxDate) {
				maxDate = f.Modified
			}
		}
		lastModified = maxDate

		return nil
	}

	if err := getDirStat(dirPath); err != nil {
		return []file{}, 0, 0, 0, time.Time{}, err
	}
	return files, totalFiles, totalSize, totalSubdirs, lastModified, nil
}
