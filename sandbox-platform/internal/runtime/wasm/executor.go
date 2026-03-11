package wasm

import (
	"encoding/json"
	"os/exec"
)

// marshalInput serialises the job input map to a JSON string.
// Returns "{}" on marshalling failure rather than panicking.
func marshalInput(input map[string]any) string {
	b, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// exitCode extracts the process exit code after cmd.Run() has been called.
func exitCode(cmd *exec.Cmd) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}
