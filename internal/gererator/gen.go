package gererator

import (
	"fmt"
	"os"
	"path"

	"github.com/emicklei/proto"
)

type service struct {
	Rpc    []rpc
	Events map[string]string
}

type rpc struct {
	Name     string
	Request  string
	Response string
	Comments []string
}

var Log llog

func Generate(fProto string, outFile string) error {
	r, err := os.Open(fProto)
	if err != nil {
		Log.Error(err)
		os.Exit(1)
	}
	defer r.Close()

	parser := proto.NewParser(r)
	definition, err := parser.Parse()
	if err != nil {
		return err
	}

	p := NewParser(fProto, path.Base(outFile))

	// step 1: collect messages
	proto.Walk(definition,
		proto.WithMessage(p.handleMessage),
	)

	// step 2: walk over RPCs
	proto.Walk(definition,
		proto.WithService(p.handleService),
		proto.WithImport(p.handleImport),
		proto.WithPackage(p.handlePackage),
		proto.WithRPC(p.handleRPC),
	)

	// write output
	Log.Printf("writing output file: %s", outFile)
	createFromTemplate(outFile, p, mainTpl)
	return nil
}

func getSubMessageName(parent, name string) string {
	return fmt.Sprintf("%s_%s", parent, name)
}

type llog struct {
	IsDebugEnabled bool
}

func (l llog) Error(err error) {
	fmt.Println(err)
}

func (l llog) Printf(s string, a ...interface{}) {
	fmt.Printf(s, a...)
}

func (l llog) Debugf(s string, a ...interface{}) {
	if l.IsDebugEnabled {
		fmt.Printf(s, a...)
	}
}

func (l llog) Debug(s ...interface{}) {
	if l.IsDebugEnabled {
		fmt.Println(s...)
	}
}

func (l llog) Println(s ...interface{}) {
	fmt.Println(s...)
}

func (l llog) Errorf(format string, s ...interface{}) {
	fmt.Printf(format, s...)
}

func (l llog) Warn(s ...interface{}) {
	fmt.Println(s...)
}

func (l llog) Info(s ...interface{}) {
	fmt.Println(s...)
}

func (l llog) Fatalln(s ...interface{}) {
	fmt.Println(s...)
	os.Exit(1)
}

func (l llog) Fatalf(format string, s ...interface{}) {
	fmt.Printf(format, s...)
	os.Exit(1)
}
