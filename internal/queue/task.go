package queue

type TaskAction int32

const (
	AddFile    = 0
	UpdateFile = 1
	DeleteFile = 2
	AddDir     = 3
	DeleteDir  = 4
)

type Task struct {
	Action TaskAction
	From   string
	To     string
}
