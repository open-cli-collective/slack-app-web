package main

import (
	"os"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/root"
)

func main() {
	os.Exit(root.Execute())
}
