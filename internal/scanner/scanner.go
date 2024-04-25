package scanner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
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

func (s *Scanner) watch(t string) {

	var toWatch map[string]Directory
	var toWatchTarget string
	//var toSync map[string]Directory
	var toSyncTarget string
	var toWatchOverwrite func()

	switch t {
	case "primary":
		toWatch = s.primaryDirectories
		//toSync = s.secondaryDirectories
		toSyncTarget = s.secondary
		toWatchTarget = s.primary
		toWatchOverwrite = func() {
			s.ScanPrimary()
		}
	case "secondary":
		toWatch = s.secondaryDirectories
		//toSync = s.primaryDirectories
		toWatchTarget = s.primary
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
			// директория удалена
			delete(toWatch, tmpDir.path)
		}

		if !tmpDir.Equals(&dir) {
			changed = true
			// обновился файл
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

			// появился новый файл
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

			// файл удалён
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

			// создана новая директория
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
	log.Println("Initial sync...")
	log.Println("Watching", s.primary)
	for true {
		s.watch("primary")
		// s.watch("secondary")
		time.Sleep(time.Nanosecond)
	}
}

func (s *Scanner) ScanSecondary() {

	directories, err := s.collectDirectories(s.primary)
	if err != nil {
		log.Panicln(err)
	}
	for _, dir := range directories {
		s.primaryDirectories[dir.path] = dir
	}

	log.Printf("Secondary directories: %d", len(s.primaryDirectories))

	var wg sync.WaitGroup

	for _, dir := range directories {
		wg.Add(1)
		d := s.primaryDirectories[dir.path]
		go s.Info(&wg, d)
	}

	wg.Wait()
	log.Println("Secondary directory scan finished")
}

func (s *Scanner) Info(wg *sync.WaitGroup, dd Directory) {
	defer wg.Done()
	dd.Info()
	s.primaryDirectories[dd.path] = dd
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
