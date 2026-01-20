package state

import "sync"

// State tracks resources created during a run to enable cleanup.
type State struct {
	mu sync.Mutex
	StateData
}

// StateData is a point-in-time copy of the monitored resources.
type StateData struct {
	VMs       []string
	Templates []string
	TempFiles []string
	Ports     []int
}

// AddVM records a VM name.
func (s *State) AddVM(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.VMs = append(s.VMs, name)
}

// AddTemplate records a template name.
func (s *State) AddTemplate(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Templates = append(s.Templates, name)
}

// AddTempFile records a temporary file path.
func (s *State) AddTempFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TempFiles = append(s.TempFiles, path)
}

// AddPort records a host port used during the run.
func (s *State) AddPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Ports = append(s.Ports, port)
}

// Snapshot returns a copy of the tracked state.
func (s *State) Snapshot() StateData {
	s.mu.Lock()
	defer s.mu.Unlock()
	return StateData{
		VMs:       append([]string{}, s.VMs...),
		Templates: append([]string{}, s.Templates...),
		TempFiles: append([]string{}, s.TempFiles...),
		Ports:     append([]int{}, s.Ports...),
	}
}
