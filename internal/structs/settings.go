package structs

type Settings struct {
	MainDir string
	SyncDir string
}

func NewSettings() *Settings {
	settings := Settings{}
	return &settings
}
