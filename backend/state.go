package backend

// BackendState stores global state shared by whole backend, i.e by all frontend connections and running process
// handlers. It consists of structs containing synchronized methods, to allow access from different goroutines.
type BackendState struct {
	messages  BackendMessages
	processes RunningProcesses
	displays  Displays
}
