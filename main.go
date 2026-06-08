package main

import "github.com/nedlane/cctrack/cmd"

func main() {
	cmd.WebFSFunc = WebFS
	cmd.Execute()
}
