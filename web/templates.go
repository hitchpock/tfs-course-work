package web

import (
	"html/template"
	"log"
	"path/filepath"

	"gitlab.com/hitchpock/tfs-course-work/internal/robot"
)

var Templates map[string]*template.Template //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	if Templates == nil {
		Templates = make(map[string]*template.Template)
	}

	filePrefix, err := filepath.Abs("/home/anton/projects/go/src/gitlab.com/hitchpock/tfs-course-work/web/template")
	if err != nil {
		log.Fatalf("path to templates not correct: %s\n", err)
		return
	}

	// Templates["wsrobotdetail"] = template.Must(template.New("").Funcs(template.FuncMap{
	// 	"validTime": func(t robot.NullTime) string {
	// 		return t.ViewHTML()
	// 	},
	// }).ParseFiles(filePrefix+"/wsrobotdetail.html", filePrefix+"/base.html"))

	Templates["listrobots"] = template.Must(template.New("").Funcs(template.FuncMap{
		"validTime": func(t robot.NullTime) string {
			return t.ViewHTML()
		},
	}).ParseFiles(filePrefix+"/listrobots.html", filePrefix+"/base.html"))

	Templates["robotdetail"] = template.Must(template.New("").Funcs(template.FuncMap{
		"validTime": func(t robot.NullTime) string {
			return t.ViewHTML()
		},
	}).ParseFiles(filePrefix+"/robotdetail.html", filePrefix+"/base.html"))
}
