package main

import (
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"golang.org/x/term"
)

// Reset sets the terminal back to the default color
var Reset = "\033[0m"

// Red sets the following text to red
var Red = "\033[31m"

// Green sets the following text to green
var Green = "\033[32m"

// The spinner draws on stdout and the log lines go to stderr, so each stream is checked
// separately. An ioctl is used rather than os.ModeCharDevice because the latter is also true for
// /dev/null, which would fill logs with escape sequences and carriage returns.
var (
	stdoutIsTerminal = term.IsTerminal(int(os.Stdout.Fd()))
	stderrIsTerminal = term.IsTerminal(int(os.Stderr.Fd()))
)

func init() {
	// Windows doesn't support console color's so remove them, and pipes or service managers
	// should not receive escape sequences either.
	if runtime.GOOS == "windows" || !stderrIsTerminal {
		Reset = ""
		Red = ""
		Green = ""
	} else {
		// Log will default override the current console line drawn by the spinner.
		log.SetPrefix("\r")
	}
}

const spinnerFrameRate = 120 * time.Millisecond

var spinnerCharset = []string{"|", "/", "-", "\\"}

// spinner is a tiny terminal spinner. It is a no-op when stdout is not a terminal, and it always
// clears itself so log output is never interleaved with a half drawn frame.
type spinner struct {
	once    sync.Once
	done    chan struct{}
	stopped chan struct{}
	msg     string
}

// startSpinner draws a spinner next to msg until Stop is called.
func startSpinner(msg string) *spinner {
	s := &spinner{
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
		msg:     msg,
	}

	if !stdoutIsTerminal {
		close(s.stopped)
		return s
	}

	go func() {
		defer close(s.stopped)
		ticker := time.NewTicker(spinnerFrameRate)
		defer ticker.Stop()

		for {
			for _, char := range spinnerCharset {
				select {
				case <-s.done:
					clearLine()
					return
				case <-ticker.C:
					writeLine(char + " " + s.msg)
				}
			}
		}
	}()

	return s
}

// Stop halts the spinner and waits for it to clear itself from the terminal.
func (s *spinner) Stop() {
	s.once.Do(func() {
		close(s.done)
	})
	<-s.stopped
}

func writeLine(line string) {
	_, _ = os.Stdout.WriteString("\r\033[2K" + line)
}

// clearLine removes whatever the spinner last drew, leaving the cursor at the start of the line.
func clearLine() {
	writeLine("")
}
