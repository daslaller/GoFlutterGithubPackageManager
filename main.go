package main

import (
	"os"

	"github.com/daslaller/GoFlutterGithubPackageManager/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
