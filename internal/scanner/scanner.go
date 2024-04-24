package scanner

import (
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

func (s *Scanner) watch(t string) {

	var toWatch map[string]Directory

	switch t {
	case "primary":
		toWatch = s.primaryDirectories
	case "secondary":
		toWatch = s.secondaryDirectories
	default:
		log.Fatalln("Invalid type given")
	}

	keys := make([]string, 0, len(toWatch))
	for key := range toWatch {
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
				for path, dir := range toWatch {
					tmpDir := Directory{
						path: path,
					}
					tmpDir.Info()

					if !tmpDir.Equals(&dir) {
						// изменилось ли количество файлов
						if tmpDir.tfiles == dir.tfiles && tmpDir.tdirectories == dir.tdirectories {
							log.Println("Updated files: ", tmpDir.GetUpdateFiles(&dir))
						}

						// появился новый файл
						if tmpDir.tfiles > dir.tfiles {
							log.Println("Added files: ", tmpDir.GetNewFiles(&dir))
						}

						// файл удалён
						if tmpDir.tfiles < dir.tfiles {
							log.Println("Deleted files: ", tmpDir.GetRemovedFiles(&dir))
						}

						// создана новая директория
						if tmpDir.tdirectories > dir.tdirectories {
							log.Println("Added directories: ")
						}

						// директория удалена
						if tmpDir.tfiles < dir.tfiles {
							log.Println("Removed directories")
						}

						toWatch[tmpDir.path] = tmpDir
					}
					time.Sleep(time.Millisecond)
				}
				time.Sleep(1 * time.Second)
			}
		}(chunk)
	}

	wg.Wait()
}

func (s *Scanner) Run() {
	log.Println("Scanning...")
	s.ScanPrimary()
	log.Println("Initial sync...")
	log.Println("Watching", s.primary)
	s.watch("primary")
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
