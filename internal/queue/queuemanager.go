package queue

import (
	"log"
	"os"
	"path/filepath"
)

type QueueManager struct {
	locked bool
	Tasks  []Task
}

func (q *QueueManager) Add(task Task) {
	q.Tasks = append(q.Tasks, task)
}

func (q *QueueManager) Run() {
	for _, t := range q.Tasks {
		switch t.Action {
		case AddFile:
			if err := copyFile(t.From, t.To); err != nil {
				log.Fatalln(err, t.From, t.To)
			}
		case UpdateFile:
			if err := copyFile(t.From, t.To); err != nil {
				log.Fatalln(err)
			}
		case DeleteFile:
			if err := removeFile(t.To); err != nil {
				log.Fatalln(err)
			}
		case AddDir:
		case DeleteDir:
		default:
			log.Panicln("Invalid action type given")
		}
	}
	q.Tasks = q.Tasks[:0]
}

func removeFile(dest string) error {
	return os.Remove(dest)
}

func copyFile(source string, dest string) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	log.Println("Copy: ", source, dest)
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	err = os.WriteFile(dest, content, 0644)
	if err != nil {
		return err
	}

	return nil
}
