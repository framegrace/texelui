package main

import (
	"flag"
	"log"

	"github.com/framegrace/texelui/apps/texelui-demo"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/standalone"
)

func main() {
	flag.Parse()
	standalone.Register("texelui-demo", func(args []string) (core.App, error) {
		return texeluidemo.New(), nil
	})
	if err := standalone.RunApp("texelui-demo", flag.Args()); err != nil {
		log.Fatalf("texelui-demo: %v", err)
	}
}
