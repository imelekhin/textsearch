package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
)

type fieldsHashTable map[string]*regexListEntryPoint

type regexListEntryPoint struct {
	next *regexListEntryPoint
	r    regexp.Regexp
	msg  string
}

type configEntry struct {
	SearchField string
	FilePath    string
	Msg         string
}

func parseConfigString(cs string) configEntry {
	var ce configEntry
	return ce
}

func loadSearches(filename string) (*fieldsHashTable, error) {

	var ce configEntry
	res := make(fieldsHashTable, 0)

	file, err := os.Open(filename)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for scanner.Scan() {
		ce = parseConfigString(scanner.Text())

	}

}
