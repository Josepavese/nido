package ops

// RequestSpawnMsg requests a VM spawn.
type RequestSpawnMsg struct {
	Name     string
	Source   string
	IsFile   bool
	UserData string
	GUI      bool
}

// RequestCreateTemplateMsg requests a template creation.
type RequestCreateTemplateMsg struct {
	Name   string
	Source string
}
