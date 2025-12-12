//go:build windows

package ansi

import (
	"golang.org/x/sys/windows"
)

// terminalState holds the original terminal state for restoration
type terminalState struct {
	mode uint32
}

// makeRaw puts the terminal into raw mode and returns the previous state
func makeRaw(fd uintptr) (*terminalState, error) {
	var mode uint32
	handle := windows.Handle(fd)

	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return nil, err
	}

	oldState := &terminalState{mode: mode}

	// Disable line input, echo, and processed input
	rawMode := mode &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_INPUT)
	// Enable virtual terminal input for escape sequences
	rawMode |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT

	if err := windows.SetConsoleMode(handle, rawMode); err != nil {
		return nil, err
	}

	return oldState, nil
}

// restoreTerminal restores the terminal to its previous state
func restoreTerminal(fd uintptr, state *terminalState) error {
	if state == nil {
		return nil
	}
	handle := windows.Handle(fd)
	return windows.SetConsoleMode(handle, state.mode)
}
