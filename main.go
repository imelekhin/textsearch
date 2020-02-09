package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"regexp"
	"strings"
)

type fieldsHashTable map[string]*searchList

type searchList struct {
	regexps    []regexp.Regexp
	commentIdx []int
	comments   []string
}

var fields fieldsHashTable

var logger *log.Logger

func splitLine(line string, linenum int) ([]string, error) {
	res := []string{"", ""}
	str := strings.TrimSpace(line[1:])

	if string([]rune(str)[0]) != ":" {
		logger.Printf("Line %d: Error: no \":\" literal between command and regexp", linenum)
		return res, errors.New("")
	}
	str = strings.TrimSpace(str[1:])

	if string([]rune(str)[0]) != "'" {
		logger.Printf("Line %d: Error: no starting ' ", linenum)
		return res, errors.New("")
	}

	str1 := strings.Split(str[1:], "'")[0]
	str2 := strings.TrimSpace(strings.Split(str[1:], "'")[1])

	if string([]rune(str2)[0]) != ":" {
		logger.Printf("Line %d: Error: no \":\" literal between regexp and comment", linenum)
		return res, errors.New("")
	}
	str2 = strings.TrimSpace(str2[1:])
	res[0] = str1
	res[1] = str2
	return res, nil
}

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
	line := 1

	fieldset := false
	currentfield := ""

	for scanner.Scan() {
		currline := strings.TrimSpace(scanner.Text())

		if len(currline) <= 1 {
			line++

		} else {

			switch string([]rune(currline)[0]) {
			case "#":
				//logger.Printf("Line %d: this is comment line", line)
				line++
				break
			case "r":
				if !fieldset {
					logger.Printf("Line %d: error: no field set", line)
				} else {
					//logger.Printf("Line %d: this is single regexp", line)
					res, err := splitLine(currline, line)
					if err != nil {
						break
					}
					rgp, err := regexp.Compile(res[0])
					if err != nil {
						logger.Printf("Line %d : Cannot compile regexp", line)
						break
					} else {
						fields[currentfield].regexps = append(fields[currentfield].regexps, *rgp)
						fields[currentfield].comments = append(fields[currentfield].comments, res[1])
						fields[currentfield].commentIdx = append(fields[currentfield].commentIdx, len(fields[currentfield].comments)-1)
					}

				}
				line++
				break
			case "f":
				if !fieldset {
					logger.Printf("Line %d: error: no field set", line)

				} else {
					//logger.Printf("Line %d: this is file", line)

				}
				line++
				break
			case "[":
				//logger.Printf("Line %d: this is start of field definition", line)
				fieldset = true
				currentfield = strings.TrimSpace(strings.Split(currline[1:], "]")[0])
				if len(currentfield) == 0 {
					logger.Printf("Line %d: Error: empty field", line)
					line++
					break
				}

				fields[currentfield] = new(searchList)

				line++
				break

			}
		}

		/*str, err := regexp.Compile(scanner.Text())
		if err != nil {
			logger.Printf("Error compiling regexp in line %d", line)
			line++
			break
		}
		res = append(res, *str)
		line++ */

	}

	return res
}

func init() {
	logger = log.New(os.Stdout, "ipsearch: ", log.Ldate|log.Ltime|log.Lshortfile)
	//comments = make([]string, 1)
	//comments[0] = "Bad regexp found"
	fields = make(fieldsHashTable, 0)

}

func main() {
	loadSearches("strings.txt")

	for k, j := range fields {
		logger.Print("Field is: ", k)
		for i := 0; i < (len(j.regexps)); i++ {
			logger.Print("On regexp '", j.regexps[i].String(), "' comment is ", j.comments[i], " line ", i)
		}

	}

}
