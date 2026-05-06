package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/robert-nix/ansihtml"
)

//go:embed lang/crash/en-us.json
var crashLangEnUS []byte

type crashPageLabels struct {
	Title           string `json:"title"`
	Heading         string `json:"heading"`
	CreateIssue     string `json:"createIssue"`
	FullLog         string `json:"fullLog"`
	Copy            string `json:"copy"`
	Copied          string `json:"copied"`
	Normal          string `json:"normal"`
	Sanitized       string `json:"sanitized"`
	SanitizedNotice string `json:"sanitizedNotice"`
	ErrorLabel      string `json:"errorLabel"`
}

func defaultCrashLabels() crashPageLabels {
	return crashPageLabels{
		Title:           "Crash report",
		Heading:         "Crash report for omnifin",
		CreateIssue:     "Create an Issue",
		FullLog:         "Full Log",
		Copy:            "Copy",
		Copied:          "Copied.",
		Normal:          "Normal",
		Sanitized:       "Sanitized",
		SanitizedNotice: "An attempt has been made to remove sensitive info, but make sure to check yourself.",
		ErrorLabel:      "Error:",
	}
}

func fillCrashLabels(s crashPageLabels) crashPageLabels {
	d := defaultCrashLabels()
	if s.Title == "" {
		s.Title = d.Title
	}
	if s.Heading == "" {
		s.Heading = d.Heading
	}
	if s.CreateIssue == "" {
		s.CreateIssue = d.CreateIssue
	}
	if s.FullLog == "" {
		s.FullLog = d.FullLog
	}
	if s.Copy == "" {
		s.Copy = d.Copy
	}
	if s.Copied == "" {
		s.Copied = d.Copied
	}
	if s.Normal == "" {
		s.Normal = d.Normal
	}
	if s.Sanitized == "" {
		s.Sanitized = d.Sanitized
	}
	if s.SanitizedNotice == "" {
		s.SanitizedNotice = d.SanitizedNotice
	}
	if s.ErrorLabel == "" {
		s.ErrorLabel = d.ErrorLabel
	}
	return s
}

func loadCrashLabels() crashPageLabels {
	def := defaultCrashLabels()
	var doc struct {
		Strings crashPageLabels `json:"strings"`
	}
	if json.Unmarshal(crashLangEnUS, &doc) != nil {
		return def
	}
	return fillCrashLabels(doc.Strings)
}

// https://gist.github.com/swdunlop/9629168
func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(4, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

// OpenFile attempts to open a given file in the appropriate GUI application.
func OpenFile(fpath string) (err error) {
	switch PLATFORM {
	case "linux":
		err = exec.Command("xdg-open", fpath).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", fpath).Start()
	case "darwin":
		err = exec.Command("open", fpath).Start()
	default:
		err = fmt.Errorf("unknown os")
	}
	return
}

// Exit dumps the last 100 lines of output to a crash file in /tmp (or equivalent), and generates a prettier HTML file containing it that is opened in the browser if possible.
func Exit(err interface{}) {
	tmpl, err2 := template.ParseFS(localFS, "html/crash.html", "html/header.html")
	if err2 != nil {
		log.Fatalf("Failed to load template: %v", err)
	}
	logCache := lineCache.String()
	if err != nil {
		fmt.Println(err)
		logCache += "\n" + fmt.Sprint(err)
	}
	logCache += "\n" + string(debug.Stack())
	sanitized := sanitizeLog(logCache)

	errStr := ""
	if err != nil {
		errStr = fmt.Sprintf("%s %v", identifyPanic(), err)
	}

	data := map[string]interface{}{
		"Log":               template.HTML(string(ansihtml.ConvertToHTML([]byte(logCache)))),
		"SanitizedLog":      template.HTML(string(ansihtml.ConvertToHTML([]byte(sanitized)))),
		"Err":               template.HTML(string(ansihtml.ConvertToHTML([]byte(errStr)))),
		"crash":             loadCrashLabels(),
		"pages":             PagePathsDTO{},
		"cssVersion":        cssVersion,
		"emailEnabled":      false,
		"discordEnabled":    false,
		"telegramEnabled":   false,
		"matrixEnabled":     false,
		"notifications":     false,
		"ombiEnabled":       false,
		"jellyseerrEnabled": false,
		"referralsEnabled":  false,
		"pwrEnabled":        false,
		"shortLang":         "en",
		"pageDirection":     "ltr",
	}
	// Use dashes for time rather than colons for Windows
	fpath := filepath.Join(temp, "omnifin-crash-"+time.Now().Local().Format("2006-01-02T15-04-05"))
	err2 = os.WriteFile(fpath+".txt", []byte(logCache), 0666)
	if err2 != nil {
		log.Fatalf("Failed to write crash dump file: %v", err2)
	}
	log.Printf("\n------\nA crash report has been saved to \"%s\".\n------", fpath+".txt")

	f, err2 := os.OpenFile(fpath+".html", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err2 != nil {
		log.Fatalf("Failed to open crash dump file: %v", err2)
	}
	defer f.Close()
	err2 = tmpl.Execute(f, data)
	if err2 != nil {
		log.Fatalf("Failed to execute crash template: %v", err2)
	}
	if err := OpenFile(fpath + ".html"); err != nil {
		log.Printf("Failed to open browser, trying text file...")
		OpenFile(fpath + ".txt")
	}
	if TRAY {
		QuitTray()
	} else {
		os.Exit(1)
	}
}
