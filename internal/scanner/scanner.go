package scanner

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
	"wslfilesync/m/v2/internal/queue"
)

type Scanner struct {
	primary   string
	secondary string

	primaryChanged   bool
	secondaryChanged bool

	primaryDirectories   map[string]Directory
	secondaryDirectories map[string]Directory

	queue queue.QueueManager
}

func (s *Scanner) initSync() {
	entries, err := os.ReadDir(s.secondary)
	if err != nil {
		log.Fatalln("Unable to read dir", err)
	}
	for _, f := range entries {
		err = os.RemoveAll(s.secondary + "/" + f.Name())
		if err != nil {
			log.Panicln("Unable to del files", err)
		}
	}
	err = copyDirContents(s.primary, s.secondary)
	if err != nil {
		log.Fatalln("Unable to copy from a to b", err)
	}
	s.ScanSecondary()
}

func (s *Scanner) watch(t string) {

	var toWatch map[string]Directory
	var toWatchTarget string
	var toSyncTarget string
	var toWatchOverwrite func()

	switch t {
	case "primary":
		toWatch = s.primaryDirectories
		toSyncTarget = s.secondary
		toWatchTarget = s.primary
		toWatchOverwrite = func() {
			s.ScanPrimary()
		}
	case "secondary":
		toWatch = s.secondaryDirectories
		toWatchTarget = s.secondary
		toSyncTarget = s.primary
		toWatchOverwrite = func() {
			s.ScanSecondary()
		}
	default:
		log.Fatalln("Invalid type given")
	}

	changed := false

	for path, dir := range toWatch {
		tmpDir := Directory{
			path: path,
		}
		if !tmpDir.Info() {
			fmt.Println("Directory removed", tmpDir.path)
			// folder deleted
			delete(toWatch, tmpDir.path)
		}

		if !tmpDir.Equals(&dir) {
			changed = true
			// file updated
			if tmpDir.tfiles == dir.tfiles && tmpDir.tdirectories == dir.tdirectories {
				updatedFiles := tmpDir.GetUpdateFiles(&dir)
				for _, f := range updatedFiles {
					s.queue.Add(queue.Task{
						Action: queue.UpdateFile,
						From:   f,
						To:     fmt.Sprintf("%s/%s/%s", toSyncTarget, tmpDir.GetRelative(toWatchTarget), filepath.Base(f)),
					})
				}
				log.Println("Updated files: ", updatedFiles)
			}

			// file added
			if tmpDir.tfiles > dir.tfiles {
				log.Println("Added files: ", tmpDir.GetNewFiles(&dir))
				addedFiles := tmpDir.GetNewFiles(&dir)
				for _, f := range addedFiles {
					s.queue.Add(queue.Task{
						Action: queue.AddFile,
						From:   f,
						To:     fmt.Sprintf("%s/%s/%s", toSyncTarget, tmpDir.GetRelative(toWatchTarget), filepath.Base(f)),
					})
				}
			}

			// file deleted
			if tmpDir.tfiles < dir.tfiles {
				deletedFiles := tmpDir.GetRemovedFiles(&dir)
				for _, f := range deletedFiles {
					s.queue.Add(queue.Task{
						Action: queue.DeleteFile,
						From:   f,
						To:     fmt.Sprintf("%s/%s/%s", toSyncTarget, tmpDir.GetRelative(toWatchTarget), filepath.Base(f)),
					})
				}
				log.Println("Deleted files: ", tmpDir.GetRemovedFiles(&dir))
			}

			// folder created
			if tmpDir.tdirectories > dir.tdirectories {
				log.Println("Added directories: ", tmpDir.GetNewDirectories(&dir))
			}
		}
		time.Sleep(time.Nanosecond)
	}

	if changed {
		s.queue.Run()
		toWatchOverwrite()
	}
}

func (s *Scanner) Run() {
	log.Println("Scanning...")
	s.ScanPrimary()
	s.ScanSecondary()
	fmt.Println(s.primaryDirectories)
	fmt.Println(s.secondaryDirectories)
	log.Println("Initial sync...")
	s.initSync()
	log.Println("Watching", s.primary)
	for true {
		s.watch("primary")
		s.watch("secondary")
		time.Sleep(time.Nanosecond)
	}
}

func (s *Scanner) ScanSecondary() {

	directories, err := s.collectDirectories(s.secondary)

	firstDir := Directory{path: s.secondary}
	firstDir.Info()

	s.secondaryDirectories[s.secondary] = firstDir

	if err != nil {
		log.Panicln(err)
	}
	for _, dir := range directories {
		s.secondaryDirectories[dir.path] = dir
	}

	log.Printf("Secondary directories: %d", len(s.secondaryDirectories))
}

func (s *Scanner) ScanPrimary() {

	directories, err := s.collectDirectories(s.primary)

	firstDir := Directory{path: s.primary}
	firstDir.Info()

	s.primaryDirectories[s.primary] = firstDir

	if err != nil {
		log.Panicln(err)
	}
	for _, dir := range directories {
		s.primaryDirectories[dir.path] = dir
	}

	log.Printf("Primary directories: %d", len(s.primaryDirectories))
}

func (s *Scanner) collectDirectories(path string) ([]Directory, error) {

	var directories []Directory

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			dir := Directory{
				path: fullPath,
			}
			dir.Info()
			directories = append(directories, dir)
			recursiveDirectories, err := s.collectDirectories(fullPath)
			if err != nil {
				log.Println(err)
				continue
			}
			directories = append(directories, recursiveDirectories...)
		}
	}
	return directories, nil
}

func copyDirContents(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirContents(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

func NewScanner(primary string, secondary string) *Scanner {

	scanner := Scanner{
		primary:              primary,
		secondary:            secondary,
		primaryDirectories:   make(map[string]Directory),
		secondaryDirectories: make(map[string]Directory),
		queue:                queue.QueueManager{},
	}
	return &scanner
}
