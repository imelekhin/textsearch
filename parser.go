package main

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
)

var opa = map[string]int{
	"&": 4,
	"|": 3,
}

type oType int

const (
	eq oType = iota
	noteq
)

type operand struct {
	field    string
	expr     []regexp.Regexp
	operator oType
}

type Cond []interface{}

func ParseInfix(e string) (rpn Cond, err error) {
	var stack []string // holds operators and left parenthesis
	for _, tok := range strings.Fields(e) {
		switch tok {
		case "(":
			stack = append(stack, tok) // push "(" to stack
		case ")":
			var op string
			for {
				// pop item ("(" or operator) from stack
				op, stack = stack[len(stack)-1], stack[:len(stack)-1]
				if op == "(" {
					break // discard "("
				}
				rpn = append(rpn, op) // add operator to result
			}
		default:
			if o1, isOp := opa[tok]; isOp {
				// token is an operator
				for len(stack) > 0 {
					// consider top item on stack
					op := stack[len(stack)-1]
					if o2, isOp := opa[op]; !isOp || o1 > o2 {
						break
					}
					// top item is an operator that needs to come off
					stack = stack[:len(stack)-1] // pop it
					rpn = append(rpn, op)        // add it to result
				}
				// push operator (the new one) to stack
				stack = append(stack, tok)
			} else { // token is an operand
				o, e := parseOperand(tok)
				if e != nil {
					return rpn, e
				}
				rpn = append(rpn, *o) // add operand to result

			}
		}
	}
	// drain stack to result
	for len(stack) > 0 {
		rpn = append(rpn, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}
	return rpn, nil
}

func (rpn Cond) Eval(m map[string]interface{}) (bool, []string) {
	var stack []bool
	var results []string

	for _, tok := range rpn {
		switch reflect.TypeOf(tok).String() {
		case "string":
			switch tok.(string) {
			case "&":
				stack[len(stack)-2] = stack[len(stack)-2] && stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			case "|":
				stack[len(stack)-2] = stack[len(stack)-2] || stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}

		default:
			res, str := tok.(operand).eval(m)
			stack = append(stack, res)
			results = append(results, str)

		}

	}

	return stack[0], results
}

func parseOperand(tok string) (*operand, error) {

	ids := strings.Split(tok, "=")
	if len(ids) != 2 {
		logger.Print("Error parsing rule ", tok)
		return nil, errors.New(tok)
	}

	op := eq

	if ids[0][len(ids[0])-1:] == "!" {
		op = noteq
		ids[0] = ids[0][0 : len(ids[0])-1]
	}

	regstr, found := expr[ids[1]]

	if !found {
		return nil, errors.New("Variable not declared " + tok)
	}

	o1 := new(operand)
	o1.field = ids[0]
	o1.operator = op

	for _, r := range regstr {

		rgp, err := regexp.Compile(r)

		if err != nil {
			return nil, err
		}

		o1.expr = append(o1.expr, *rgp)

	}

	return o1, nil

}

func (o operand) eval(m map[string]interface{}) (bool, string) {
	var found bool = false
	var str string

	for field, value := range m {
		if field == o.field {
			for i := 0; i < len(o.expr); i++ {
				str = o.expr[i].FindString(value.(string))
				if len(str) > 0 {
					found = true
					break

				}

			}

			switch o.operator {
			case eq:
				break
			case noteq:
				found = !found
			}
		}

	}

	return found, str

}
