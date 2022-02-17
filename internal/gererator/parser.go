package gererator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emicklei/proto"
)

type (
	Parser struct {
		Pkg            string
		CurrentService string
		Services       map[string]*service
		// map<messageName, *messagesStorItem>
		MessagesStor map[string]*messagesStorItem
		// map<messageName, map <enumName, *enum>>
		MessageEnums map[string]map[string]*proto.Enum
		Filepath     string
		Filename     string
	}
	messagesStorItem struct {
		msg *proto.Message
		// map<fieldName, map<optionName, *option>>
		fieldOptions map[string]map[string]*proto.Option
	}
)

func NewParser(protoFilePath string, outFileName string) *Parser {
	return &Parser{
		MessagesStor: map[string]*messagesStorItem{},
		MessageEnums: map[string]map[string]*proto.Enum{},
		Services:     map[string]*service{},
		Filepath:     protoFilePath,
		Filename:     outFileName,
	}
}

func (g *Parser) handleImport(p *proto.Import) {
	dir := filepath.Dir(g.Filepath)
	path := filepath.Join(dir, p.Filename)
	r, err := os.Open(path)

	if err != nil {
		Log.Error(err)
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

func (g *Parser) handlePackage(p *proto.Package) {
	g.Pkg = p.Name
}

func (g *Parser) handleService(s *proto.Service) {
	g.Services[s.Name] = &service{
		Rpc:    []rpc{},
		Events: map[string]string{},
	}
	messages := map[string]bool{}
	g.CurrentService = s.Name
	for _, v := range g.MessagesStor {
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
			Log.Debugf("event %s: %s\n", v, match[0][1])

			re := regexp.MustCompile(`\s*[,]+\s*`)
			strs := re.Split(match[0][1], -1)
			_ = strs
			for _, event := range strs {
				topic := event

				// Find custom topics
				parts := strings.SplitN(event, ":", 2)
				if len(parts) == 2 {
					topic = parts[1]
					event = parts[0]
				}

				if !messages[event] {
					fmt.Printf("Error while parsing proto. wrong message name for event \"%s\" in servcie %s", topic, s.Name)
					os.Exit(1)
				}
				if events[event] {
					fmt.Printf("Error event \"%s\" already discribed in service %s", topic, s.Name)
					os.Exit(1)
				}
				Log.Debugf("event: %v topic: %v", event, topic)
				events[event] = true
				g.Services[s.Name].Events[topic] = event
			}
		}
	}
}

func (g *Parser) handleRPC(prpc *proto.RPC) {
	// todo: document?

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

func (g *Parser) handleMessage(m *proto.Message) {
	Log.Debugf("message: %s", m.Name)

	// collect options
	fieldOptionsMap := map[string]map[string]*proto.Option{}
	for _, f := range m.Elements {
		optionsMap := map[string]*proto.Option{}
		if nff, ok := f.(*proto.NormalField); ok {
			for _, o := range nff.Options {
				Log.Debugf("message %s option: %s = %s",
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
	g.MessagesStor[m.Name] = &messagesStorItem{
		msg:          m,
		fieldOptions: fieldOptionsMap,
	}

	// walk fields to collect sub-messages (only 1 level deep)
	for _, f := range m.Elements {
		if msg, ok := f.(*proto.Message); ok {
			subMsgName := getSubMessageName(m.Name, msg.Name)
			g.MessagesStor[subMsgName] = &messagesStorItem{
				msg:          msg,
				fieldOptions: map[string]map[string]*proto.Option{},
			}

		} else if enum, ok := f.(*proto.Enum); ok {
			if len(g.MessageEnums[m.Name]) == 0 {
				g.MessageEnums[m.Name] = map[string]*proto.Enum{}
			}
			g.MessageEnums[m.Name][enum.Name] = enum
		}
	}
}
