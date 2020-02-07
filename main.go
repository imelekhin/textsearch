package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
)

type fieldsHashTable map[string]searchList

type searchList struct {
	regexps  []regexp.Regexp
	comments []string
}

var fields fieldsHashTable

var logger *log.Logger

func loadSearches(filename string) []regexp.Regexp {

	res := make([]regexp.Regexp, 0)

	file, err := os.Open(filename)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	line := 0
	for scanner.Scan() {
		str, err := regexp.Compile(scanner.Text())
		if err != nil {
			logger.Printf("Error compiling regexp in line %d", line)
			line++
			break
		}
		res = append(res, *str)
		line++

	}

}

func init() {
	logger = log.New(os.Stdout, "ipsearch: ", log.Ldate|log.Ltime|log.Lshortfile)
	comments = make([]string, 1)
	comments[0] = "Bad regexp found"
	fields = make(fieldsHashTable, 0)

}
