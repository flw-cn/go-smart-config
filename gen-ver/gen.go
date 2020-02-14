package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/flw-cn/go-smartConfig"
)

type Contributor struct {
	Name  string
	Lines int
}

type Config struct {
	App      string `flag:"a|foo|your application's name"`
	Package  string `flag:"P|main|the name of the package where version.go is located"`
	FileName string `flag:"f|version.go|the name of the generated code file"`
	Path     string `flag:"p|.|the path to place the generated code file"`
}

func main() {
	config := Config{}
	smartConfig.LoadConfig("Version Number Generator", "v1.0", &config)

	authorList := parseContributors(RunCommand(`git log --stat`))
	appVersion := RunCommand(`git describe --always --tags --dirty`)
	buildHost := RunCommand(`hostname`)
	goVersion := RunCommand(`go version`)

	fileName := filepath.Join(config.Path, config.FileName)

	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf(`os.Create("%s"): %s`, fileName, err)
		return
	}

	now := time.Now()

	err = fileTemplate.Execute(file, struct {
		Timestamp      time.Time
		Carls          []Contributor
		Package        string
		AppName        string
		Version        string
		BuildGoVersion string
		BuildHost      string
		GoVersion      string
	}{
		Timestamp: now,
		Carls:     authorList,
		Package:   config.Package,
		AppName:   config.App,
		Version:   appVersion,
		BuildHost: buildHost,
		GoVersion: goVersion,
	})

	if err != nil {
		log.Printf("text/template.Template.Execute() returns error: %s", err)
		return
	}
}

func parseContributors(gitLog string) []Contributor {
	var author string

	authorDict := make(map[string]int)
	lines := strings.Split(gitLog, "\n")

	for _, line := range lines {
		fields := strings.SplitN(line, " ", 2)
		if fields[0] == "Author:" {
			author = fields[1]
			continue
		}

		re := regexp.MustCompile(` (\d+) insertion`)

		subs := re.FindStringSubmatch(line)
		if subs != nil {
			lines, _ := strconv.Atoi(subs[1])
			authorDict[author] += lines
		}
	}

	authorList := []Contributor{}

	for k, v := range authorDict {
		authorList = append(authorList, Contributor{
			Name:  k,
			Lines: v,
		})
	}

	sort.SliceStable(authorList, func(i, j int) bool {
		return authorList[i].Lines > authorList[j].Lines
	})

	return authorList
}

// #nosec
func RunCommand(cmdLine string) string {
	args := regexp.MustCompile(`\s+`).Split(cmdLine, -1)
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()

	if err != nil {
		log.Fatal(cmdLine, ": ", err)
	}

	return strings.Trim(string(output), "\r\n\t ")
}

var fileTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at {{ .Timestamp }}

package {{ printf "%s" .Package }}

var Contributors = []struct {
	Name  string
	Lines int
}{
{{- range .Carls }}
	{{ printf "{%q, %d}" .Name .Lines }},
{{- end }}
}

var (
	AppName        = {{ printf "%q" .AppName }}
	Version        = {{ printf "%q" .Version }}
	BuildTime      = {{.Timestamp.Format "2006-01-02 15:04:05 MST" | printf "%q"}}
	BuildGoVersion = {{ printf "%q" .GoVersion }}
	BuildHost      = {{ printf "%q" .BuildHost }}
)
`))
