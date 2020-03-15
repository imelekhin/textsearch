package main

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/cloudflare/ahocorasick"
)

type rule struct {
	name      string        //rule name
	alarm     string        //text to rise alarm
	condition Cond          //expression in reverse polish notation with variables name
	action    int           //reserved
	execTime  time.Duration //for speed profiling
	execCount int64         //for speed profiling
}

// list of supported logic operations
var opa = map[string]int{
	"&": 4,
	"|": 3,
}

type oType int // operation type - equal or not equal

const (
	eq oType = iota
	noteq
)

type operand struct { //atomic operand
	field    string //message field name to apply operand
	expr     string //variable name to calculate
	operator oType  // operator: field equal or not equal variable
}

type Cond []interface{} //

type variables map[string][]string //hash map variable name -> string with regexp or []string with patterns loaded from file

type variablecompiled struct { //compiled variable exression: compiled regexp or ahocorasick matcher
	name string //variable name for search
	v    interface{}
}

type variables1 map[string]variablecompiled //compiled variables hash map variable name to compiled variable

func (e variablecompiled) Eval(s string) (bool, string) { //check s against variable expression regexp or ahocorasick
	var found bool = false
	var str string
	switch reflect.TypeOf(e.v).String() {
	case "*regexp.Regexp":
		str = e.v.(*regexp.Regexp).FindString(s)
		if len(str) > 0 {
			found = true
		}

	case "*ahocorasick.Matcher":
		matches := e.v.(*ahocorasick.Matcher).Match([]byte(s))
		if len(matches) > 0 {
			found = true
			str = expr[e.name][matches[0]] //bad part shoud be refactored

		}

	}
	return found, str
}

func ParseInfix(e string, expr variables) (rpn Cond, err error) { //parse rule condition from string to reverse polish notation
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
				o, e := parseOperand(tok, expr)
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

func (rpn Cond) Eval(m map[string]interface{}) (bool, []string) { //calculate value for expression in reverse polish notation
	var stack []interface{} //bool
	var results []string

	if len(rpn) == 1 {
		r, s := rpn[0].(operand).eval(m)
		results = append(results, s)
		return r, results
	}

	for _, tok := range rpn {
		switch reflect.TypeOf(tok).String() {
		case "string":
			var res1, res2 bool
			var str string
			switch tok.(string) {
			case "&":
				if reflect.TypeOf(stack[len(stack)-2]).String() != "bool" { //some optimization tricks in logic operations
					res1, str = stack[len(stack)-2].(operand).eval(m)
					results = append(results, str)
				} else {
					res1 = stack[len(stack)-2].(bool)
				}
				res2 = false
				if res1 {
					if reflect.TypeOf(stack[len(stack)-1]).String() != "bool" { //some optimization tricks in logic operations
						res2, str = stack[len(stack)-1].(operand).eval(m)
						results = append(results, str)
					} else {
						res2 = stack[len(stack)-1].(bool)
					}

				}
				stack[len(stack)-2] = res1 && res2
				stack = stack[:len(stack)-1]
			case "|":
				if reflect.TypeOf(stack[len(stack)-2]).String() != "bool" { //some optimization tricks in logic operations
					res1, str = stack[len(stack)-2].(operand).eval(m)
					results = append(results, str)
				} else {
					res1 = stack[len(stack)-2].(bool)
				}
				res2 = false
				if !res1 {
					if reflect.TypeOf(stack[len(stack)-1]).String() != "bool" { //some optimization tricks in logic operations
						res2, str = stack[len(stack)-1].(operand).eval(m)
						results = append(results, str)
					} else {
						res2 = stack[len(stack)-1].(bool)
					}

				}

				stack[len(stack)-2] = res1 || res2
				stack = stack[:len(stack)-1]
			}

		default:
			//res, str := tok.(operand).eval(m)
			stack = append(stack, tok) ////res)
			//results = append(results, str)

		}

	}

	return stack[0].(bool), results
}

func parseOperand(tok string, expr variables) (*operand, error) {

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

	_, found := expr[ids[1]]

	if !found {
		return nil, errors.New("Variable not declared " + tok)
	}

	o1 := new(operand)
	o1.field = ids[0]
	o1.operator = op
	o1.expr = ids[1]

	return o1, nil

}

func (o operand) eval(m map[string]interface{}) (bool, string) {
	var found bool = false
	var str string

	for field, value := range m {
		if field == o.field {

			found, str = varscompiled[o.expr].Eval(value.(string))

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
