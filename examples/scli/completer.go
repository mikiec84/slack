package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
)

var commands = []string{
	"c", "c-archive", "c-create", "c-history", "c-info", "c-invite", "c-join", "c-kick", "c-leave", "c-list", "c-rename", "c-purpose", "c-topic", "c-unarchive",
	"g", "g-archive", "g-close", "g-create", "g-createChild", "g-history", "g-info", "g-invite", "g-kick", "g-leave", "g-list", "g-open", "g-rename", "g-purpose", "g-topic", "g-unarchive",
	"d", "d-close", "d-history", "d-list", "d-open",
	"f-delete", "f-info", "f-list", "f", "f-c",
	"r-add", "r-get", "r-list", "r-remove",
	"s", "s-files", "s-messages",
	"t-info", "t-logs",
	"u-presence", "u-info", "u-list",
	"m-delete", "m-update",
	"e-list",
	"hist", "purpose", "topic", "list",
}

// toIntf converts a slice or array of a specific type to array of interface{}
func toIntf(s interface{}) []interface{} {
	v := reflect.ValueOf(s)
	// There is no need to check, we want to panic if it's not slice or array
	intf := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		intf[i] = v.Index(i).Interface()
	}
	return intf
}

// in checks if val is in s slice
func in(s interface{}, val interface{}) bool {
	si := toIntf(s)
	for _, v := range si {
		if v == val {
			return true
		}
	}
	return false
}

func findCompletions(line []rune, pos int, parts, names []string) (head string, completions []string, tail string) {
	l := len(parts)
	endsWithSpace := pos > 0 && unicode.IsSpace(line[pos-1])
	for _, name := range names {
		if endsWithSpace {
			if !in(parts, name) {
				completions = append(completions, name)
			}
		} else if strings.HasPrefix(name, parts[l-1]) {
			if !in(parts[:l-1], name) {
				completions = append(completions, name)
			}
		}
	}
	if endsWithSpace {
		head = string(line[:pos])
	} else {
		head = string(line[:pos-len([]rune(parts[l-1]))])
	}
	if pos < len(line) {
		tail = string(line[pos:])
	}
	return
}

func channelCompletions(line []rune, pos int, parts []string) (head string, completions []string, tail string) {
	var names []string
	for i := range info.Channels {
		names = append(names, info.Channels[i].Name)
	}
	return findCompletions(line, pos, parts, names)
}

func groupCompletions(line []rune, pos int, parts []string) (head string, completions []string, tail string) {
	var names []string
	for i := range info.Groups {
		names = append(names, info.Groups[i].Name)
	}
	return findCompletions(line, pos, parts, names)
}

func imCompletions(line []rune, pos int, parts []string) (head string, completions []string, tail string) {
	var names []string
	for i := range info.IMS {
		names = append(names, findUser(info.IMS[i].User).Name)
	}
	return findCompletions(line, pos, parts, names)
}

func userCompletions(line []rune, pos int, parts []string) (head string, completions []string, tail string) {
	var names []string
	for i := range info.Users {
		names = append(names, info.Users[i].Name)
	}
	return findCompletions(line, pos, parts, names)
}

func fileCompletions(line []rune, pos int, parts []string) (head string, completions []string, tail string) {
	var names []string
	for i := range files {
		names = append(names, files[i].Name)
	}
	return findCompletions(line, pos, parts, names)
}

func osFileCompletions(line []rune, pos int, parts []string) (head string, completions []string, tail string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working dir - %v", err)
		return
	}
	file := ""
	// if this is not a new file we are starting with
	if !unicode.IsSpace(line[pos-1]) || strings.HasSuffix(string(line), "\\ ") {
		lineCopy := strings.Replace(string(line[:pos]), "\\ ", " ", -1)
		index := strings.Index(lineCopy, " ")
		for index != -1 {
			if line[index-1] != '\\' {
				lineCopy = lineCopy[index+1:]
				index = strings.Index(lineCopy, " ")
			} else {
				index = strings.Index(lineCopy[index+1:], " ")
			}
		}
		if lineCopy[0] == os.PathSeparator || len(lineCopy) > 2 && lineCopy[1] == ':' {
			dir, file = filepath.Split(lineCopy)
		} else {
			dir, file = filepath.Split(fmt.Sprintf("%s%c%s", dir, os.PathSeparator, lineCopy))
		}
	}
	dirFile, err := os.Open(dir)
	if err != nil {
		return
	}
	fi, err := dirFile.Readdir(-1)
	if err != nil {
		return
	}
	for i := range fi {
		if strings.HasPrefix(fi[i].Name(), file) {
			name := strings.Replace(fi[i].Name(), " ", "\\ ", -1)
			if fi[i].IsDir() {
				name = fmt.Sprintf("%s%c", name, os.PathSeparator)
			}
			completions = append(completions, name)
		}
	}
	if pos < len(line) {
		tail = string(line[pos:])
	}
	head = string(line[:pos-len([]rune(file))])
	return
}

func textCompletions(runes []rune, pos int) (head string, completions []string, tail string) {
	i := pos - 1
	for i >= 0 {
		if unicode.IsSpace(runes[i]) {
			break
		}
		if runes[i] == '@' {
			for j := range info.Users {
				if strings.HasPrefix(info.Users[j].Name, string(runes[i+1:pos])) {
					completions = append(completions, info.Users[j].Name)
				}
			}
			head = string(runes[:i+1])
			tail = string(runes[pos:])
		}
		i--
	}
	return
}

func completer(line string, pos int) (head string, completions []string, tail string) {
	runes := []rune(line)
	prefix := string(runes[:pos])
	if !strings.HasPrefix(prefix, Options.CommandPrefix) {
		return textCompletions(runes, pos)
	}
	parts := strings.Fields(prefix)
	l := len(parts)
	endsWithSpace := pos > 0 && unicode.IsSpace(runes[pos-1])
	// we are trying to complete command
	if l == 1 && !endsWithSpace {
		for _, c := range commands {
			cmd := Options.CommandPrefix + c
			if strings.HasPrefix(cmd, parts[0]) {
				completions = append(completions, c)
			}
		}
		head = Options.CommandPrefix
		if pos < len(runes) {
			tail = string(runes[pos:])
		}
	} else {
		cmd := strings.ToLower(parts[0][len(Options.CommandPrefix):])
		switch cmd {
		case "c-archive", "c-create", "c-info", "c-join", "c-leave", "c-unarchive":
			return channelCompletions(runes, pos, parts[1:])
		case "c", "c-history", "c-invite", "c-kick", "c-rename", "c-purpose", "c-topic":
			// Since if len is 1 then it has to end with space if we are here
			if l == 1 || l == 2 && !endsWithSpace {
				return channelCompletions(runes, pos, parts[1:])
			} else if cmd == "c-invite" || cmd == "c-kick" {
				if l == 2 && endsWithSpace || l >= 3 && !endsWithSpace {
					return userCompletions(runes, pos, parts[2:])
				}
			}
		case "g-archive", "g-close", "g-create", "g-createChild", "g-info", "g-leave", "g-open", "g-unarchive":
			return groupCompletions(runes, pos, parts[1:])
		case "g", "g-history", "g-invite", "g-kick", "g-rename", "g-purpose", "g-topic":
			// Since if len is 1 then it has to end with space if we are here
			if l == 1 || l == 2 && !endsWithSpace {
				return groupCompletions(runes, pos, parts[1:])
			} else if cmd == "g-invite" || cmd == "g-kick" {
				if l == 2 && endsWithSpace || l >= 3 && !endsWithSpace {
					return userCompletions(runes, pos, parts[2:])
				}
			}
		case "d-close", "d-open":
			return imCompletions(runes, pos, parts[1:])
		case "d", "d-history", "d-list":
			// Since if len is 1 then it has to end with space if we are here
			if l == 1 || l == 2 && !endsWithSpace {
				return imCompletions(runes, pos, parts[1:])
			}
		case "f-delete", "f-info", "f-c":
			return fileCompletions(runes, pos, parts[1:])
		case "f":
			return osFileCompletions(runes, pos, parts[1:])
		}
	}
	return
}
