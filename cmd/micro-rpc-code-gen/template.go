package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

//go:embed template.tmpl
var mainTpl string

// TODO: what to do with context???

func createFromTemplate(generatedFilePath string, data interface{}, tpl string) {
	//a, _ := json.Marshal(data)
	//fmt.Println("done", string(a))
	//os.Exit(1)
	if _, err := os.Stat(filepath.Dir(generatedFilePath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(generatedFilePath), 0755); err != nil {
			log.Fatalln(err)
		}
	}

	f, e := os.Create(generatedFilePath)
	if e != nil {
		log.Fatalf(`file "%s" create error: %s`, generatedFilePath, e)
		return
	}

	o := f
	defer func() {
		err := f.Close()
		if err != nil {
			log.Errorf("output file close error: %s", err)
		}
	}()

	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"Escape": func(s string) string {
			return strings.ReplaceAll(
				strings.ReplaceAll(s, `\`, `\\"`),
				`"`,
				`\"`)
		},
		"Title":   strings.Title,
		"ToLower": strings.ToLower,
	}

	// TODO: add some code generation for registry; add listen to rpc direct commands

	if t, e := template.New("").Funcs(funcMap).Parse(tpl); e != nil {
		log.Fatalln(fmt.Errorf(`template parse error: %s`, e))
	} else if e := t.Execute(o, data); e != nil {
		log.Fatalln(fmt.Errorf(`template execute error: %s`, e))
	}
}