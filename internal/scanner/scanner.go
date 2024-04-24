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

func Sync() {

}

func (s *Scanner) Run() {
	s.ScanPrimary()
	//s.ScanSecondary()

	/*var chunkSize = 10
	if len(s.primaryDirectories) < 10 {
		chunkSize = len(s.primaryDirectories)
	}*/

	//for _, value := range s.primaryDirectories {
	//fmt.Println(value)
	//}

	keys := make([]string, 0, len(s.primaryDirectories))
	for key := range s.primaryDirectories {
		keys = append(keys, key)
	}

	chunkSize := len(keys) / 10
	chunks := make([][]string, 10)
	for i := 0; i < 10; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == 9 {
			end = len(keys)
		}
		chunks[i] = keys[start:end]
	}

	var wg sync.WaitGroup

	for _, chunk := range chunks {
		wg.Add(1)
		go func(dirs []string) {
			defer wg.Done()
			for {
				for path, dir := range s.primaryDirectories {
					tmpDir := Directory{
						path: path,
					}
					tmpDir.Info()

					if !tmpDir.Equals(&dir) {
						// изменилось ли количество файлов
						if tmpDir.tfiles == dir.tfiles {
							fmt.Println("Updated files: ", tmpDir.GetUpdateFiles(&dir))
						}

						// появился новый файл
						if tmpDir.tfiles > dir.tfiles {
							fmt.Println("Added files: ", tmpDir.GetNewFiles(&dir))
						}

						// файл удалён
						if tmpDir.tfiles < dir.tfiles {
							fmt.Println("Deleted files: ", tmpDir.GetRemovedFiles(&dir))
						}

						// создана новая директория
						if tmpDir.tdirectories > dir.tdirectories {

						}

						// директория удалена
						if tmpDir.tfiles < dir.tfiles {

						}

						s.primaryDirectories[tmpDir.path] = tmpDir
					}
					time.Sleep(time.Millisecond)
				}
				time.Sleep(1 * time.Second)
			}
		}(chunk)
	}

	wg.Wait()

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
	log.Println("Secondray directory scan finished")
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
	log.Println("Primary directory scan finished")
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
