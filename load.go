package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
)

// ToDo  - add error checks
func Load(f string, expr variables) error {

	file, err := os.Open(f)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	line := 1

	for scanner.Scan() {
		currline := strings.TrimSpace(scanner.Text())
		if len(currline) <= 1 {
			line++
		} else {
			tokens := strings.Fields(currline)
			switch tokens[0] {
			case "var":
				for scanner.Scan() {
					currline := strings.TrimSpace(scanner.Text())
					line++
					for len(currline) <= 1 {
						scanner.Scan()
						currline = strings.TrimSpace(scanner.Text())
						line++
					}

					if currline == "endvar" {
						break
					}
					t := strings.Split(scanner.Text(), "=")
					if len(t) != 2 {
						return errors.New("Not valid variable definition at line : " + strconv.Itoa(line))
					}
					expr[strings.TrimSpace(t[0])] = append(expr[strings.TrimSpace(t[0])], strings.TrimSpace(t[1]))

				}

			case "list":
				for scanner.Scan() {
					currline := strings.TrimSpace(scanner.Text())
					line++
					for len(currline) <= 1 {
						scanner.Scan()
						currline = strings.TrimSpace(scanner.Text())
						line++
					}

					if currline == "endlist" {
						break
					}
					t := strings.Split(scanner.Text(), "=")
					if len(t) != 2 {
						return errors.New("Not valid variable definition at line : " + strconv.Itoa(line))
					}
					expr[strings.TrimSpace(t[0])], err = loadRegexpFromFile(strings.TrimSpace(t[1]))
					if err != nil {
						return err
					}

				}

			case "rule":
				line++
				if len(tokens) < 2 {
					return errors.New("Not valid rule name : " + strconv.Itoa(line))
				}

				r := new(rule)
				r.execTime = 0
				r.execCount = 0
				r.name = tokens[1]

				for scanner.Scan() {
					currline = strings.TrimSpace(scanner.Text())
					line++
					for len(currline) <= 1 { //skip empty lines
						scanner.Scan()
						currline = strings.TrimSpace(scanner.Text())
						line++
					}

					if currline == "endrule" {
						break
					}

					ruletokens := strings.SplitN(currline, " ", 2)

					switch ruletokens[0] {
					case "if":
						r.condition, err = ParseInfix(strings.TrimSpace(ruletokens[1]))
						if err != nil {
							return err
						}
					case "alarm":
						r.alarm = strings.TrimSpace(ruletokens[1])
					default:
						return errors.New("Not valid comand inside rule : " + strconv.Itoa(line))
					}

				}

				rulelist = append(rulelist, *r)

			}

		}

	}

	return nil

}

func loadRegexpFromFile(filename string) ([]string, error) {

	var res []string
	file, err := os.Open(filename)

	defer file.Close()

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for scanner.Scan() {
		str := strings.TrimSpace(scanner.Text())
		if len(str) >= 1 {
			res = append(res, str)
		}

	}

	return res, nil
}
