package docker

type ProjectStatus int

const (
	AllRunning ProjectStatus = iota
	SomeStopped
	AllStopped
	Unknown
)

type ComposeProject struct {
	Name       string
	Containers []Container
	ConfigFile string        // from label
	WorkingDir string        // from label
	Status     ProjectStatus // all running, some stopped, etc
}

// Container holds all the data we show in the TUI
type Container struct {
	ID     string   // short container id
	Names  []string // can have multiple names
	Image  string   // image name like "nginx:latest"
	Status string   // human readable status
	State  string   // running/exited/etc
	Memory string   // mem usage %
	CPU    string   // cpu usage %
	//PIDs    string // process count
	Ports          string // ports
	NetIO          string // network I/O
	BlockIO        string // block I/O
	ComposeProject string // compose project name (empty if standalone)
	ComposeService string // compose service name
	ComposeNumber  string // compose container number
}
type ComposeInfo struct {
	Project string
	Service string
	Number  int
}

// ContainerStats holds stats for a single container
type ContainerStats struct {
	ID     string
	CPU    string
	Memory string
	// PIDs    string
	NetIO   string
	BlockIO string
}

// sent when we finish fetching the container list
type ContainersMsg struct {
	Containers []Container
	Err        error
}

// sent when logs are ready
type LogsMsg struct {
	ID    string
	Lines []string
	Err   error
}
