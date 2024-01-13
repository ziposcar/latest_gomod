package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Line struct {
	line int
	text string
}

type ModVersion struct {
	line      Line
	path      string
	major     int64
	minor     int64
	patch     int64
	timestamp *string
	hash      *string
}

func (modVersion *ModVersion) getTimestamp() string {
	if modVersion.timestamp == nil {
		return ""
	}
	return *modVersion.timestamp
}

func (modVersion *ModVersion) getHash() string {
	if modVersion.hash == nil {
		return ""
	}
	return *modVersion.hash
}

func later(a, b *ModVersion) bool {
	if a.major != b.major {
		return a.major > b.major
	}
	if a.minor != b.minor {
		return a.minor > b.minor
	}
	if a.patch != b.patch {
		return a.patch > b.patch
	}
	if a.getTimestamp() != b.getTimestamp() {
		return a.getTimestamp() > b.getTimestamp()
	}
	if a.getHash() != b.getHash() {
		return a.getHash() > b.getHash()
	}
	return false
}

func main() {
	file, err := os.Open("go.mod")
	if err != nil {
		panic(err)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	file.Close()
	goModContent := strings.Split(string(content), "\n")

	requiring := false

	requires := []Line{}
	toBeDeleteLines := map[int]Line{}

	for line, text := range goModContent {
		line := Line{
			line: line,
			text: text,
		}
		if _, _, conflict := getConflictPart(line); conflict {
			toBeDeleteLines[line.line] = line
			continue
		}
		if strings.TrimSpace(line.text) == "require (" {
			requiring = true
			continue
		}
		if strings.TrimSpace(line.text) == ")" {
			requiring = false
			continue
		}
		if requiring {
			requires = append(requires, line)
		}
	}

	latestSet := map[string]*ModVersion{}
	for _, line := range requires {
		if strings.TrimSpace(line.text) == "" {
			continue
		}
		modVersion := getModVersionByLine(line)
		if modVersion == nil {
			continue
		}
		latestModVersion := latestSet[modVersion.path]
		if latestModVersion == nil {
			latestSet[modVersion.path] = modVersion
			continue
		}
		if later(modVersion, latestModVersion) {
			latestSet[modVersion.path] = modVersion
			toBeDeleteLines[latestModVersion.line.line] = latestModVersion.line
		} else {
			toBeDeleteLines[modVersion.line.line] = modVersion.line
		}
	}

	newGoModContent := make([]string, 0, len(goModContent))
	for i, line := range goModContent {
		if _, ok := toBeDeleteLines[i]; ok {
			continue
		}
		newGoModContent = append(newGoModContent, line)
	}
	newContent := strings.Join(newGoModContent, "\n")
	_ = os.WriteFile("go.mod", []byte(newContent), 0644)
	cmd := exec.Command("go", "mod", "tidy")
	println(cmd.String())
	_ = cmd.Run()
}

func getConflictPart(line Line) (bool, bool, bool) {
	var p1, p2 = false, false
	lineText := strings.TrimSpace(line.text)
	if strings.HasPrefix(lineText, "<<<<<<<") {
		p1, p2 = true, false
	} else if lineText == "=======" {
		p1, p2 = false, true
	} else if strings.HasPrefix(lineText, ">>>>>>>") {
		p1, p2 = false, false
	} else {
		return false, false, false
	}
	return p1, p2, true
}

func getModVersionByLine(line Line) *ModVersion {
	v := &ModVersion{
		line: line,
	}
	lineTexts := strings.Split(strings.TrimSpace(line.text), " ")
	if lineTexts[0] != "" {
		v.path = lineTexts[0]
	}
	modText := lineTexts[1]
	if modText[0] != 'v' {
		return nil
	}
	modList := strings.Split(modText[1:], "-")
	if len(modList) > 1 {
		v.timestamp = &modList[1]
	}
	if len(modList) > 2 {
		v.hash = &modList[2]
	}
	version := modList[0]
	versionList := strings.Split(version, ".")
	v.major, _ = strconv.ParseInt(versionList[0], 10, 64)
	v.minor, _ = strconv.ParseInt(versionList[1], 10, 64)
	v.patch, _ = strconv.ParseInt(versionList[2], 10, 64)

	return v
}
