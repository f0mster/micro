package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"
)

const appTitle = "RPC code generator (c) Rendall Labs 2020"

var version = "unknown"

func main() {
	fmt.Println(appTitle)
	fmt.Println("Version:", version)

	fDebug := flag.Bool("d", false, "debug mode")
	fVerboseDebug := flag.Bool("dd", false, "more verbose debug mode")
	fProto := flag.String("proto", "", "path to proto file")
	foutPath := flag.String("out", "", "output path")
	flag.Parse()

	if *fDebug || *fVerboseDebug {
		log.Info("debug mode")
		log.SetLevel(log.DebugLevel)

		if *fVerboseDebug {
			log.SetReportCaller(true)
			formatter := &log.TextFormatter{
				CallerPrettyfier: func(f *runtime.Frame) (string, string) {
					return fmt.Sprintf("%s()", f.Function),
						fmt.Sprintf(" %s:%d", path.Base(f.File), f.Line)
				},
			}
			log.SetFormatter(formatter)
		}
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if *fProto == "" {
		log.Warn("-proto flag must be used")
		os.Exit(1)
	}
	var out string
	if *foutPath == "" {
		out = *fProto + ".rpc.go"
	} else {
		out = path.Join(*foutPath, path.Base(*fProto)+".rpc.go")
	}
	err := Generate(*fProto, out)
	if err != nil {
		log.Errorf("gen error: %s", err)
	}

	log.Println("done.")
}
