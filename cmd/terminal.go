package main

import (
	"log"
	"runtime"
)

// Reset sets the terminal back to the default color
var Reset = "\033[0m"

// Red sets the following text to red
var Red = "\033[31m"

// Green sets the following text to green
var Green = "\033[32m"

func init() {
	// Log will be default override the current console line
	log.SetPrefix("\r")

	// Windows doesn't support console color's so remove them
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
	}
}
