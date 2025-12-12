//go:build !windows

package ansi

import (
	"golang.org/x/sys/unix"
)

// terminalState holds the original terminal state for restoration
type terminalState struct {
	termios unix.Termios
}

// makeRaw puts the terminal into raw mode and returns the previous state
func makeRaw(fd uintptr) (*terminalState, error) {
	termios, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	if err != nil {
		return nil, err
	}

	oldState := &terminalState{termios: *termios}

	// Set raw mode flags
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(int(fd), unix.TCSETS, termios); err != nil {
		return nil, err
	}

	return oldState, nil
}

// restoreTerminal restores the terminal to its previous state
func restoreTerminal(fd uintptr, state *terminalState) error {
	if state == nil {
		return nil
	}
	return unix.IoctlSetTermios(int(fd), unix.TCSETS, &state.termios)
}
