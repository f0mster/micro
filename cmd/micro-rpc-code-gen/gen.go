package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/emicklei/proto"
	log "github.com/sirupsen/logrus"
)

type (
	generator struct {
		Pkg            string
		CurrentService string
		Services       map[string]*service
		// map<messageName, *messagesStorItem>
		messagesStor map[string]*messagesStorItem
		// map<messageName, map <enumName, *enum>>
		messageEnums map[string]map[string]*proto.Enum
		filepath     string
	}
	messagesStorItem struct {
		msg *proto.Message
		// map<fieldName, map<optionName, *option>>
		fieldOptions map[string]map[string]*proto.Option
	}
)

type service struct {
	Rpc    []rpc
	Events []string
}

type rpc struct {
	Name     string
	Request  string
	Response string
	Comments []string
}

func Generate(fProto string, outFile string) error {
	r, err := os.Open(fProto)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	defer r.Close()

	parser := proto.NewParser(r)
	definition, err := parser.Parse()
	if err != nil {
		return err
	}

	g := &generator{
		messagesStor: map[string]*messagesStorItem{},
		messageEnums: map[string]map[string]*proto.Enum{},
		Services:     map[string]*service{},
		filepath:     fProto,
	}

	// step 1: collect messages
	proto.Walk(definition,
		proto.WithMessage(g.handleMessage),
	)

	// step 2: walk over RPCs
	proto.Walk(definition,
		proto.WithService(g.handleService),
		proto.WithImport(g.handleImport),
		proto.WithPackage(g.handlePackage),
		proto.WithRPC(g.handleRPC),
	)

	// write output
	log.Printf("writing output file: %s", outFile)
	createFromTemplate(outFile, g, mainTpl)
	return nil
}

func (g *generator) handleImport(p *proto.Import) {
	dir := filepath.Dir(g.filepath)
	path := filepath.Join(dir, p.Filename)
	r, err := os.Open(path)

	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	defer r.Close()
	parser := proto.NewParser(r)
	definition, err := parser.Parse()
	if err != nil {
		panic(fmt.Errorf("parser error: %w", err))
	}
	proto.Walk(definition,
		proto.WithMessage(g.handleMessage),
	)
}

func (g *generator) handlePackage(p *proto.Package) {
	g.Pkg = p.Name
}

func (g *generator) handleService(s *proto.Service) {
	g.Services[s.Name] = &service{
		Rpc:    []rpc{},
		Events: []string{},
	}
	g.Services[s.Name].Events = make([]string, 0)
	messages := map[string]bool{}
	g.CurrentService = s.Name
	for _, v := range g.messagesStor {
		messages[v.msg.Name] = true
	}
	events := map[string]bool{}
	if s.Comment == nil {
		return
	}
	for _, v := range s.Comment.Lines {
		re := regexp.MustCompile(`^\s*\*?\s*@event\s*(.*)\s*$`)
		match := re.FindAllStringSubmatch(v, -1)
		if len(match) >= 1 {
			log.Debugf("event %s: %s\n", v, match[0][1])

			re := regexp.MustCompile(`\s*[,]+\s*`)
			strs := re.Split(match[0][1], -1)
			_ = strs
			for _, v := range strs {
				if !messages[v] {
					fmt.Printf("Error while parsing proto. wrong message name for event \"%s\" in servcie %s", v, s.Name)
					os.Exit(1)
				}
				if events[v] {
					fmt.Printf("Error event \"%s\" already discribed in service %s", v, s.Name)
					os.Exit(1)
				}
				log.Debug("events:", v)
				events[v] = true
				g.Services[s.Name].Events = append(g.Services[s.Name].Events, v)

			}
		}
	}
}

func (g *generator) handleRPC(prpc *proto.RPC) {
	// todo: doccoment?

	myRpc := rpc{
		Name:     prpc.Name,
		Request:  prpc.RequestType,
		Response: prpc.ReturnsType,
	}

	if prpc.Comment != nil && len(prpc.Comment.Lines) > 0 {
		myRpc.Comments = make([]string, 0, len(prpc.Comment.Lines))
		myRpc.Comments = append(myRpc.Comments, prpc.Comment.Lines...)
	}

	g.Services[g.CurrentService].Rpc = append(
		g.Services[g.CurrentService].Rpc,
		myRpc,
	)
}

func getSubMessageName(parent, name string) string {
	return fmt.Sprintf("%s_%s", parent, name)
}

func (g *generator) handleMessage(m *proto.Message) {
	log.Debugf("message: %s", m.Name)

	// collect options
	fieldOptionsMap := map[string]map[string]*proto.Option{}
	for _, f := range m.Elements {
		optionsMap := map[string]*proto.Option{}
		if nff, ok := f.(*proto.NormalField); ok {
			for _, o := range nff.Options {
				log.Debugf("message %s option: %s = %s",
					m.Name,
					o.Name,
					o.Constant.Source)

				optionsMap[o.Name] = o
			}
		}
		if nff, ok := f.(*proto.NormalField); ok {
			fieldOptionsMap[nff.Name] = optionsMap
		}
	}
	// store message
	g.messagesStor[m.Name] = &messagesStorItem{
		msg:          m,
		fieldOptions: fieldOptionsMap,
	}

	// walk fields to collect sub-messages (only 1 level deep)
	for _, f := range m.Elements {
		if msg, ok := f.(*proto.Message); ok {
			subMsgName := getSubMessageName(m.Name, msg.Name)
			g.messagesStor[subMsgName] = &messagesStorItem{
				msg:          msg,
				fieldOptions: map[string]map[string]*proto.Option{},
			}

		} else if enum, ok := f.(*proto.Enum); ok {
			if len(g.messageEnums[m.Name]) == 0 {
				g.messageEnums[m.Name] = map[string]*proto.Enum{}
			}
			g.messageEnums[m.Name][enum.Name] = enum
		}
	}
}
