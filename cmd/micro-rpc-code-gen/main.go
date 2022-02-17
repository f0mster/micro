package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/f0mster/micro/internal/gererator"
)

const appTitle = "RPC code generator"

var version = "v0.2.0"

func main() {
	fmt.Println(appTitle)
	fmt.Println("Version:", version)

	fDebug := flag.Bool("d", false, "debug mode")
	fProto := flag.String("proto", "", "path to proto file")
	foutPath := flag.String("out", "", "output path")
	flag.Parse()
	gererator.Log.IsDebugEnabled = *fDebug
	if *fProto == "" {
		gererator.Log.Warn("-proto flag must be used")
		os.Exit(1)
	}
	out := path.Base(*fProto) + ".rpc.go"
	if *foutPath == "" {
		out = path.Join(*foutPath, out)
	}
	err := gererator.Generate(*fProto, out)
	if err != nil {
		gererator.Log.Errorf("gen error: %s", err)
	}

	gererator.Log.Println("done.")
}
