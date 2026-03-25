// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package fhirpath implements the FHIRPath expression language for navigating
// and extracting data from FHIR resources.
//
// Usage:
//
//	result, err := fhirpath.Evaluate(patient, "name.where(use='official').family")
//	result, err := fhirpath.Evaluate(obs, "value.is(Quantity)")
//
// Compiled expressions (evaluate many times):
//
//	expr, _ := fhirpath.Compile("name.family")
//	result1, _ := expr.Evaluate(patient1)
//	result2, _ := expr.Evaluate(patient2)
package fhirpath

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Collection is the result of evaluating a FHIRPath expression.
// In FHIRPath, everything is a collection (even single values).
type Collection []any

// Resolver resolves a FHIR reference (e.g., "Patient/123") to a resource.
// Used by the resolve() function. Return nil if the reference cannot be resolved.
type Resolver func(reference string) any

// Expression is a compiled FHIRPath expression.
type Expression struct {
	source   string
	ast      Node
	resolver Resolver
}

// Compile parses a FHIRPath expression for repeated evaluation.
func Compile(expr string) (*Expression, error) {
	ast, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	return &Expression{source: expr, ast: ast}, nil
}

// WithResolver sets a reference resolver for the resolve() function.
func (e *Expression) WithResolver(r Resolver) *Expression {
	e.resolver = r
	return e
}

// Evaluate evaluates the compiled expression against a resource.
func (e *Expression) Evaluate(resource any) (Collection, error) {
	ctx := &evalContext{
		resource: resource,
		resolver: e.resolver,
	}
	input := Collection{resource}
	return ctx.eval(e.ast, input)
}

// String returns the original expression string.
func (e *Expression) String() string { return e.source }

// Evaluate parses and evaluates a FHIRPath expression in one step.
func Evaluate(resource any, expr string) (Collection, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.Evaluate(resource)
}

// EvaluateWithResolver parses and evaluates with a reference resolver.
func EvaluateWithResolver(resource any, expr string, resolver Resolver) (Collection, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	compiled.resolver = resolver
	return compiled.Evaluate(resource)
}

// EvaluateBool evaluates an expression and returns a single boolean result.
// Returns false if the result is empty, the single boolean value if exactly
// one boolean, or an error for other cases.
func EvaluateBool(resource any, expr string) (bool, error) {
	result, err := Evaluate(resource, expr)
	if err != nil {
		return false, err
	}
	return result.Bool()
}

// Bool converts a collection to a boolean per FHIRPath rules:
// empty → false, single boolean → that value, single non-boolean → true.
func (c Collection) Bool() (bool, error) {
	if len(c) == 0 {
		return false, nil
	}
	if len(c) == 1 {
		if b, ok := c[0].(bool); ok {
			return b, nil
		}
		return true, nil // non-empty singleton is truthy
	}
	return false, fmt.Errorf("fhirpath: collection has %d items, cannot convert to boolean", len(c))
}

// String returns the first string value, or empty string.
func (c Collection) String() string {
	if len(c) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", c[0])
}

// evalContext holds state during evaluation.
type evalContext struct {
	resource any      // root resource (for %resource)
	resolver Resolver // reference resolver (for resolve())
}

func (ctx *evalContext) eval(node Node, input Collection) (Collection, error) {
	switch n := node.(type) {
	case *LiteralNode:
		return Collection{n.Value}, nil

	case *IdentNode:
		return ctx.evalIdent(n.Name, input)

	case *DotNode:
		left, err := ctx.eval(n.Left, input)
		if err != nil {
			return nil, err
		}
		return ctx.eval(n.Right, left)

	case *IndexNode:
		coll, err := ctx.eval(n.Expr, input)
		if err != nil {
			return nil, err
		}
		idxColl, err := ctx.eval(n.Index, input)
		if err != nil {
			return nil, err
		}
		if len(idxColl) == 0 {
			return nil, nil
		}
		idx := toInt(idxColl[0])
		if idx >= 0 && idx < int64(len(coll)) {
			return Collection{coll[idx]}, nil
		}
		return nil, nil

	case *FunctionNode:
		return ctx.evalFunction(n.Name, n.Args, input)

	case *UnaryNode:
		return ctx.evalUnary(n.Op, n.Expr, input)

	case *BinaryNode:
		return ctx.evalBinary(n.Op, n.Left, n.Right, input)

	case *TypeNode:
		return ctx.evalType(n.Op, n.Expr, n.TypeName, input)

	case *QuantityNode:
		return Collection{quantityValue{Value: n.Value, Unit: n.Unit}}, nil

	case *EmptyNode:
		return nil, nil

	default:
		return nil, fmt.Errorf("fhirpath: unknown node type %T", node)
	}
}

func (ctx *evalContext) evalIdent(name string, input Collection) (Collection, error) {
	// Environment variables
	switch name {
	case "%resource", "$this":
		if ctx.resource != nil {
			return Collection{ctx.resource}, nil
		}
		return input, nil
	case "%context":
		return input, nil
	}

	var result Collection
	for _, item := range input {
		vals := getField(item, name)
		result = append(result, vals...)
	}
	return result, nil
}

func (ctx *evalContext) evalUnary(op string, expr Node, input Collection) (Collection, error) {
	val, err := ctx.eval(expr, input)
	if err != nil {
		return nil, err
	}
	switch op {
	case "-":
		if len(val) == 0 {
			return nil, nil
		}
		return Collection{negate(val[0])}, nil
	case "not":
		b, err := val.Bool()
		if err != nil {
			return nil, err
		}
		return Collection{!b}, nil
	}
	return nil, fmt.Errorf("fhirpath: unknown unary op %q", op)
}

func (ctx *evalContext) evalBinary(op string, left, right Node, input Collection) (Collection, error) {
	lval, err := ctx.eval(left, input)
	if err != nil {
		return nil, err
	}

	// Short-circuit for boolean operators
	switch op {
	case "and":
		lb, _ := lval.Bool()
		if !lb && len(lval) > 0 {
			return Collection{false}, nil
		}
		rval, err := ctx.eval(right, input)
		if err != nil {
			return nil, err
		}
		rb, _ := rval.Bool()
		return Collection{lb && rb}, nil

	case "or":
		lb, _ := lval.Bool()
		if lb {
			return Collection{true}, nil
		}
		rval, err := ctx.eval(right, input)
		if err != nil {
			return nil, err
		}
		rb, _ := rval.Bool()
		return Collection{lb || rb}, nil

	case "implies":
		lb, _ := lval.Bool()
		if !lb {
			return Collection{true}, nil
		}
		rval, err := ctx.eval(right, input)
		if err != nil {
			return nil, err
		}
		rb, _ := rval.Bool()
		return Collection{rb}, nil

	case "xor":
		rval, err := ctx.eval(right, input)
		if err != nil {
			return nil, err
		}
		lb, _ := lval.Bool()
		rb, _ := rval.Bool()
		return Collection{lb != rb}, nil
	}

	rval, err := ctx.eval(right, input)
	if err != nil {
		return nil, err
	}

	switch op {
	case "=":
		return Collection{collEqual(lval, rval)}, nil
	case "!=":
		return Collection{!collEqual(lval, rval)}, nil
	case "<", "<=", ">", ">=":
		return ctx.evalCompare(op, lval, rval)
	case "+":
		return ctx.evalArith(op, lval, rval)
	case "-":
		return ctx.evalArith(op, lval, rval)
	case "*":
		return ctx.evalArith(op, lval, rval)
	case "/":
		return ctx.evalArith(op, lval, rval)
	case "mod":
		return ctx.evalArith(op, lval, rval)
	case "div":
		return ctx.evalArith(op, lval, rval)
	case "|":
		// Union
		return append(lval, rval...), nil
	case "&":
		// String concatenation
		ls := ""
		rs := ""
		if len(lval) > 0 {
			ls = fmt.Sprintf("%v", lval[0])
		}
		if len(rval) > 0 {
			rs = fmt.Sprintf("%v", rval[0])
		}
		return Collection{ls + rs}, nil
	case "in":
		if len(lval) == 0 {
			return Collection{true}, nil
		}
		for _, l := range lval {
			found := false
			for _, r := range rval {
				if valEqual(l, r) {
					found = true
					break
				}
			}
			if !found {
				return Collection{false}, nil
			}
		}
		return Collection{true}, nil
	case "contains":
		if len(rval) == 0 {
			return Collection{true}, nil
		}
		for _, r := range rval {
			found := false
			for _, l := range lval {
				if valEqual(l, r) {
					found = true
					break
				}
			}
			if !found {
				return Collection{false}, nil
			}
		}
		return Collection{true}, nil
	}

	return nil, fmt.Errorf("fhirpath: unknown binary op %q", op)
}

func (ctx *evalContext) evalCompare(op string, lval, rval Collection) (Collection, error) {
	if len(lval) == 0 || len(rval) == 0 {
		return nil, nil
	}
	cmp := compareValues(lval[0], rval[0])
	switch op {
	case "<":
		return Collection{cmp < 0}, nil
	case "<=":
		return Collection{cmp <= 0}, nil
	case ">":
		return Collection{cmp > 0}, nil
	case ">=":
		return Collection{cmp >= 0}, nil
	}
	return nil, nil
}

func (ctx *evalContext) evalArith(op string, lval, rval Collection) (Collection, error) {
	if len(lval) == 0 || len(rval) == 0 {
		return nil, nil
	}

	// Date arithmetic: date + duration quantity → new date
	if op == "+" || op == "-" {
		if result, ok := tryDateArith(op, lval[0], rval[0]); ok {
			return Collection{result}, nil
		}
	}

	// String concatenation with +
	if op == "+" {
		if _, ok := lval[0].(string); ok {
			return Collection{fmt.Sprintf("%v", lval[0]) + fmt.Sprintf("%v", rval[0])}, nil
		}
	}

	l := toFloat(lval[0])
	r := toFloat(rval[0])
	switch op {
	case "+":
		return Collection{l + r}, nil
	case "-":
		return Collection{l - r}, nil
	case "*":
		return Collection{l * r}, nil
	case "/":
		if r == 0 {
			return nil, nil // division by zero → empty
		}
		return Collection{l / r}, nil
	case "mod":
		if r == 0 {
			return nil, nil
		}
		return Collection{math.Mod(l, r)}, nil
	case "div":
		if r == 0 {
			return nil, nil
		}
		return Collection{float64(int64(l) / int64(r))}, nil
	}
	return nil, nil
}

func (ctx *evalContext) evalType(op string, expr Node, typeName string, input Collection) (Collection, error) {
	val, err := ctx.eval(expr, input)
	if err != nil {
		return nil, err
	}
	switch op {
	case "is":
		if len(val) == 0 {
			return Collection{false}, nil
		}
		return Collection{isType(val[0], typeName)}, nil
	case "as":
		var result Collection
		for _, v := range val {
			if isType(v, typeName) {
				result = append(result, v)
			}
		}
		return result, nil
	}
	return nil, nil
}

// --- Built-in functions ---

func (ctx *evalContext) evalFunction(name string, args []Node, input Collection) (Collection, error) {
	switch name {
	case "where":
		if len(args) != 1 {
			return nil, fmt.Errorf("where() requires 1 argument")
		}
		var result Collection
		for _, item := range input {
			val, err := ctx.eval(args[0], Collection{item})
			if err != nil {
				return nil, err
			}
			b, _ := val.Bool()
			if b {
				result = append(result, item)
			}
		}
		return result, nil

	case "select":
		if len(args) != 1 {
			return nil, fmt.Errorf("select() requires 1 argument")
		}
		var result Collection
		for _, item := range input {
			val, err := ctx.eval(args[0], Collection{item})
			if err != nil {
				return nil, err
			}
			result = append(result, val...)
		}
		return result, nil

	case "exists":
		if len(args) == 0 {
			return Collection{len(input) > 0}, nil
		}
		// exists(criteria)
		for _, item := range input {
			val, err := ctx.eval(args[0], Collection{item})
			if err != nil {
				return nil, err
			}
			b, _ := val.Bool()
			if b {
				return Collection{true}, nil
			}
		}
		return Collection{false}, nil

	case "empty":
		return Collection{len(input) == 0}, nil

	case "count":
		return Collection{int64(len(input))}, nil

	case "first":
		if len(input) > 0 {
			return Collection{input[0]}, nil
		}
		return nil, nil

	case "last":
		if len(input) > 0 {
			return Collection{input[len(input)-1]}, nil
		}
		return nil, nil

	case "tail":
		if len(input) > 1 {
			return input[1:], nil
		}
		return nil, nil

	case "take":
		if len(args) != 1 {
			return nil, fmt.Errorf("take() requires 1 argument")
		}
		nColl, err := ctx.eval(args[0], input)
		if err != nil {
			return nil, err
		}
		n := int(toInt(nColl[0]))
		if n > len(input) {
			n = len(input)
		}
		return input[:n], nil

	case "skip":
		if len(args) != 1 {
			return nil, fmt.Errorf("skip() requires 1 argument")
		}
		nColl, err := ctx.eval(args[0], input)
		if err != nil {
			return nil, err
		}
		n := int(toInt(nColl[0]))
		if n >= len(input) {
			return nil, nil
		}
		return input[n:], nil

	case "all":
		if len(args) != 1 {
			return nil, fmt.Errorf("all() requires 1 argument")
		}
		for _, item := range input {
			val, err := ctx.eval(args[0], Collection{item})
			if err != nil {
				return nil, err
			}
			b, _ := val.Bool()
			if !b {
				return Collection{false}, nil
			}
		}
		return Collection{true}, nil

	case "not":
		b, err := input.Bool()
		if err != nil {
			return nil, err
		}
		return Collection{!b}, nil

	case "distinct":
		var result Collection
		seen := make(map[string]bool)
		for _, item := range input {
			key := fmt.Sprintf("%v", item)
			if !seen[key] {
				seen[key] = true
				result = append(result, item)
			}
		}
		return result, nil

	case "hasValue":
		if len(input) == 0 {
			return Collection{false}, nil
		}
		return Collection{input[0] != nil}, nil

	case "iif":
		if len(args) < 2 {
			return nil, fmt.Errorf("iif() requires 2-3 arguments")
		}
		cond, err := ctx.eval(args[0], input)
		if err != nil {
			return nil, err
		}
		b, _ := cond.Bool()
		if b {
			return ctx.eval(args[1], input)
		}
		if len(args) > 2 {
			return ctx.eval(args[2], input)
		}
		return nil, nil

	// String functions
	case "startsWith":
		return ctx.stringFunc(input, args, func(s, arg string) any { return strings.HasPrefix(s, arg) })
	case "endsWith":
		return ctx.stringFunc(input, args, func(s, arg string) any { return strings.HasSuffix(s, arg) })
	case "contains":
		return ctx.stringFunc(input, args, func(s, arg string) any { return strings.Contains(s, arg) })
	case "replace":
		if len(args) != 2 || len(input) == 0 {
			return nil, nil
		}
		s := toString(input[0])
		patColl, _ := ctx.eval(args[0], input)
		repColl, _ := ctx.eval(args[1], input)
		pat := toString(patColl[0])
		rep := toString(repColl[0])
		return Collection{strings.ReplaceAll(s, pat, rep)}, nil
	case "length":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{int64(len(toString(input[0])))}, nil
	case "substring":
		if len(input) == 0 || len(args) < 1 {
			return nil, nil
		}
		s := toString(input[0])
		startColl, _ := ctx.eval(args[0], input)
		start := int(toInt(startColl[0]))
		if start >= len(s) {
			return Collection{""}, nil
		}
		if len(args) > 1 {
			lenColl, _ := ctx.eval(args[1], input)
			length := int(toInt(lenColl[0]))
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return Collection{s[start:end]}, nil
		}
		return Collection{s[start:]}, nil
	case "upper":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{strings.ToUpper(toString(input[0]))}, nil
	case "lower":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{strings.ToLower(toString(input[0]))}, nil
	case "trim":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{strings.TrimSpace(toString(input[0]))}, nil
	case "toInteger":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{toInt(input[0])}, nil
	case "toDecimal":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{toFloat(input[0])}, nil
	case "toString":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{toString(input[0])}, nil
	case "toBoolean":
		if len(input) == 0 {
			return nil, nil
		}
		s := strings.ToLower(toString(input[0]))
		return Collection{s == "true" || s == "1"}, nil

	// Type functions
	case "ofType":
		if len(args) != 1 {
			return nil, fmt.Errorf("ofType() requires 1 argument")
		}
		typeName := ""
		if ident, ok := args[0].(*IdentNode); ok {
			typeName = ident.Name
		}
		var result Collection
		for _, item := range input {
			if isType(item, typeName) {
				result = append(result, item)
			}
		}
		return result, nil

	// Type functions (also available as infix operators)
	case "is":
		if len(args) != 1 || len(input) == 0 {
			return Collection{false}, nil
		}
		typeName := ""
		if ident, ok := args[0].(*IdentNode); ok {
			typeName = ident.Name
		}
		return Collection{isType(input[0], typeName)}, nil

	case "as":
		if len(args) != 1 {
			return nil, nil
		}
		typeName := ""
		if ident, ok := args[0].(*IdentNode); ok {
			typeName = ident.Name
		}
		var result Collection
		for _, v := range input {
			if isType(v, typeName) {
				result = append(result, v)
			}
		}
		return result, nil

	// resolve() — follow a Reference
	case "resolve":
		if ctx.resolver == nil {
			return nil, nil // no resolver configured
		}
		var result Collection
		for _, item := range input {
			ref := ""
			// Get reference string from Reference object or string
			refVals := getField(item, "reference")
			if len(refVals) > 0 {
				ref = toString(refVals[0])
			} else {
				ref = toString(item)
			}
			if ref == "" {
				continue
			}
			resolved := ctx.resolver(ref)
			if resolved != nil {
				result = append(result, resolved)
			}
		}
		return result, nil

	// FHIR-specific
	case "extension":
		if len(args) != 1 {
			return nil, fmt.Errorf("extension() requires 1 argument")
		}
		urlColl, err := ctx.eval(args[0], input)
		if err != nil {
			return nil, err
		}
		url := toString(urlColl[0])
		return ctx.evalExtension(input, url)

	// Math functions
	case "abs":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{math.Abs(toFloat(input[0]))}, nil
	case "ceiling":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{math.Ceil(toFloat(input[0]))}, nil
	case "floor":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{math.Floor(toFloat(input[0]))}, nil
	case "round":
		if len(input) == 0 {
			return nil, nil
		}
		precision := 0
		if len(args) > 0 {
			pColl, _ := ctx.eval(args[0], input)
			if len(pColl) > 0 {
				precision = int(toInt(pColl[0]))
			}
		}
		factor := math.Pow(10, float64(precision))
		return Collection{math.Round(toFloat(input[0])*factor) / factor}, nil
	case "ln":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{math.Log(toFloat(input[0]))}, nil
	case "log":
		if len(input) == 0 || len(args) < 1 {
			return nil, nil
		}
		baseColl, _ := ctx.eval(args[0], input)
		base := toFloat(baseColl[0])
		return Collection{math.Log(toFloat(input[0])) / math.Log(base)}, nil
	case "exp":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{math.Exp(toFloat(input[0]))}, nil
	case "sqrt":
		if len(input) == 0 {
			return nil, nil
		}
		return Collection{math.Sqrt(toFloat(input[0]))}, nil
	case "power":
		if len(input) == 0 || len(args) < 1 {
			return nil, nil
		}
		expColl, _ := ctx.eval(args[0], input)
		return Collection{math.Pow(toFloat(input[0]), toFloat(expColl[0]))}, nil

	// Regex matches
	case "matches":
		if len(input) == 0 || len(args) < 1 {
			return nil, nil
		}
		s := toString(input[0])
		patColl, err := ctx.eval(args[0], input)
		if err != nil {
			return nil, err
		}
		pat := toString(patColl[0])
		re, err := regexp.Compile(pat)
		if err != nil {
			return Collection{false}, nil
		}
		return Collection{re.MatchString(s)}, nil

	case "replaceMatches":
		if len(input) == 0 || len(args) < 2 {
			return nil, nil
		}
		s := toString(input[0])
		patColl, _ := ctx.eval(args[0], input)
		repColl, _ := ctx.eval(args[1], input)
		re, err := regexp.Compile(toString(patColl[0]))
		if err != nil {
			return Collection{s}, nil
		}
		return Collection{re.ReplaceAllString(s, toString(repColl[0]))}, nil

	// Date/time functions
	case "now":
		return Collection{time.Now().Format(time.RFC3339)}, nil
	case "today":
		return Collection{time.Now().Format("2006-01-02")}, nil
	case "timeOfDay":
		return Collection{time.Now().Format("15:04:05")}, nil

	// Type check functions (without conversion)
	case "convertsToInteger":
		if len(input) == 0 {
			return Collection{false}, nil
		}
		_, err := strconv.ParseInt(toString(input[0]), 10, 64)
		return Collection{err == nil}, nil
	case "convertsToDecimal":
		if len(input) == 0 {
			return Collection{false}, nil
		}
		_, err := strconv.ParseFloat(toString(input[0]), 64)
		return Collection{err == nil}, nil
	case "convertsToBoolean":
		if len(input) == 0 {
			return Collection{false}, nil
		}
		s := strings.ToLower(toString(input[0]))
		return Collection{s == "true" || s == "false" || s == "1" || s == "0"}, nil
	case "convertsToString":
		return Collection{len(input) > 0}, nil

	// Collection traversal
	case "children":
		var result Collection
		for _, item := range input {
			result = append(result, allChildren(item)...)
		}
		return result, nil
	case "descendants":
		var result Collection
		for _, item := range input {
			result = append(result, allDescendants(item)...)
		}
		return result, nil

	// repeat(expr) — recursive expansion
	case "repeat":
		if len(args) != 1 {
			return nil, fmt.Errorf("repeat() requires 1 argument")
		}
		seen := make(map[string]bool)
		var result Collection
		current := input
		for len(current) > 0 {
			var next Collection
			for _, item := range current {
				key := fmt.Sprintf("%p", item)
				if seen[key] {
					continue
				}
				seen[key] = true
				result = append(result, item)
				val, err := ctx.eval(args[0], Collection{item})
				if err != nil {
					return nil, err
				}
				next = append(next, val...)
			}
			current = next
		}
		return result, nil

	// aggregate(expr, init)
	case "aggregate":
		if len(args) < 1 {
			return nil, fmt.Errorf("aggregate() requires at least 1 argument")
		}
		var total any
		if len(args) > 1 {
			initColl, _ := ctx.eval(args[1], input)
			if len(initColl) > 0 {
				total = initColl[0]
			}
		}
		// For simplicity, aggregate with $total not fully supported
		// Just sum numeric values
		for _, item := range input {
			if total == nil {
				total = item
			} else {
				total = toFloat(total) + toFloat(item)
			}
		}
		if total == nil {
			return nil, nil
		}
		return Collection{total}, nil

	// trace(name) — debugging, just passes through
	case "trace":
		return input, nil

	// Utility
	case "type":
		if len(input) == 0 {
			return nil, nil
		}
		rv := reflect.ValueOf(input[0])
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		return Collection{rv.Type().Name()}, nil

	case "single":
		if len(input) != 1 {
			return nil, nil
		}
		return input, nil

	case "indexOf":
		if len(input) == 0 || len(args) < 1 {
			return nil, nil
		}
		s := toString(input[0])
		subColl, _ := ctx.eval(args[0], input)
		sub := toString(subColl[0])
		return Collection{int64(strings.Index(s, sub))}, nil

	case "split":
		if len(input) == 0 || len(args) < 1 {
			return nil, nil
		}
		s := toString(input[0])
		sepColl, _ := ctx.eval(args[0], input)
		sep := toString(sepColl[0])
		parts := strings.Split(s, sep)
		var result Collection
		for _, p := range parts {
			result = append(result, p)
		}
		return result, nil

	case "join":
		if len(input) == 0 {
			return Collection{""}, nil
		}
		sep := ""
		if len(args) > 0 {
			sepColl, _ := ctx.eval(args[0], input)
			if len(sepColl) > 0 {
				sep = toString(sepColl[0])
			}
		}
		var parts []string
		for _, item := range input {
			parts = append(parts, toString(item))
		}
		return Collection{strings.Join(parts, sep)}, nil

	default:
		return nil, fmt.Errorf("fhirpath: unknown function %q", name)
	}
}

func (ctx *evalContext) stringFunc(input Collection, args []Node, fn func(string, string) any) (Collection, error) {
	if len(input) == 0 || len(args) < 1 {
		return nil, nil
	}
	s := toString(input[0])
	argColl, err := ctx.eval(args[0], input)
	if err != nil {
		return nil, err
	}
	arg := toString(argColl[0])
	return Collection{fn(s, arg)}, nil
}

func (ctx *evalContext) evalExtension(input Collection, url string) (Collection, error) {
	var result Collection
	for _, item := range input {
		exts := getField(item, "extension")
		for _, ext := range exts {
			urlVals := getField(ext, "url")
			for _, u := range urlVals {
				if toString(u) == url {
					result = append(result, ext)
				}
			}
		}
	}
	return result, nil
}

// tryDateArith attempts date arithmetic: date +/- duration quantity.
// Returns (result, true) if the operation was a date+duration, (nil, false) otherwise.
func tryDateArith(op string, left, right any) (string, bool) {
	// Left must be a date string, right must be a duration quantity
	dateStr := ""
	var dur quantityValue
	hasDur := false

	if s, ok := left.(string); ok && isDateLike(s) {
		dateStr = s
		if q, ok := right.(quantityValue); ok && isDurationUnit(q.Unit) {
			dur = q
			hasDur = true
		}
	}

	if !hasDur {
		return "", false
	}

	// Parse the date
	t, err := parseDate(dateStr)
	if err != nil {
		return "", false
	}

	amount := int(dur.Value)
	if op == "-" {
		amount = -amount
	}

	switch dur.Unit {
	case "year", "years":
		t = t.AddDate(amount, 0, 0)
	case "month", "months":
		t = t.AddDate(0, amount, 0)
	case "week", "weeks":
		t = t.AddDate(0, 0, amount*7)
	case "day", "days":
		t = t.AddDate(0, 0, amount)
	case "hour", "hours":
		t = t.Add(time.Duration(amount) * time.Hour)
	case "minute", "minutes":
		t = t.Add(time.Duration(amount) * time.Minute)
	case "second", "seconds":
		t = t.Add(time.Duration(amount) * time.Second)
	default:
		return "", false
	}

	// Format back to original precision
	return formatDateLike(t, dateStr), true
}

func isDurationUnit(unit string) bool {
	switch unit {
	case "year", "years", "month", "months", "week", "weeks",
		"day", "days", "hour", "hours", "minute", "minutes",
		"second", "seconds":
		return true
	}
	return false
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-01",
		"2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q", s)
}

func formatDateLike(t time.Time, original string) string {
	switch {
	case len(original) == 4: // YYYY
		return t.Format("2006")
	case len(original) == 7: // YYYY-MM
		return t.Format("2006-01")
	case len(original) == 10: // YYYY-MM-DD
		return t.Format("2006-01-02")
	case strings.Contains(original, "T") && strings.Contains(original, "Z"):
		return t.Format("2006-01-02T15:04:05Z07:00")
	case strings.Contains(original, "T"):
		return t.Format("2006-01-02T15:04:05")
	default:
		return t.Format("2006-01-02")
	}
}

// quantityValue represents a FHIRPath quantity literal (e.g., 5 'mg').
type quantityValue struct {
	Value float64
	Unit  string
}

func (q quantityValue) String() string {
	return fmt.Sprintf("%v '%s'", q.Value, q.Unit)
}

// toQuantity extracts a quantity from a value (Quantity struct or quantityValue).
func toQuantity(v any) (float64, string, bool) {
	switch q := v.(type) {
	case quantityValue:
		return q.Value, q.Unit, true
	default:
		// Try to extract from a Quantity struct via reflection
		val := getField(v, "value")
		unit := getField(v, "unit")
		if len(val) > 0 {
			u := ""
			if len(unit) > 0 {
				u = toString(unit[0])
			}
			// Also check code field for UCUM
			if u == "" {
				code := getField(v, "code")
				if len(code) > 0 {
					u = toString(code[0])
				}
			}
			return toFloat(val[0]), u, true
		}
	}
	return 0, "", false
}

// --- Reflection helpers ---

func allChildren(obj any) Collection {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	var result Collection
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" || strings.HasPrefix(strings.Split(tag, ",")[0], "_") {
			continue
		}
		result = append(result, reflectToCollection(v.Field(i))...)
	}
	return result
}

func allDescendants(obj any) Collection {
	var result Collection
	children := allChildren(obj)
	for _, child := range children {
		result = append(result, child)
		result = append(result, allDescendants(child)...)
	}
	return result
}

func getField(obj any, name string) Collection {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")

		if tag == "-" {
			// json:"-" fields are value[x] unions and Extra — match by Go name
			if strings.EqualFold(field.Name, name) {
				fv := v.Field(i)
				return reflectToCollection(fv)
			}
			continue
		}
		if tag == "" {
			continue
		}
		jsonName := strings.Split(tag, ",")[0]
		if jsonName != name && !strings.EqualFold(field.Name, name) {
			continue
		}
		if strings.HasPrefix(jsonName, "_") {
			continue
		}

		fv := v.Field(i)
		return reflectToCollection(fv)
	}
	return nil
}

func reflectToCollection(v reflect.Value) Collection {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return reflectToCollection(v.Elem())
	case reflect.Slice:
		if v.IsNil() || v.Len() == 0 {
			return nil
		}
		var result Collection
		for i := 0; i < v.Len(); i++ {
			result = append(result, reflectToCollection(v.Index(i))...)
		}
		return result
	case reflect.String:
		s := v.String()
		if s == "" {
			return nil
		}
		return Collection{s}
	case reflect.Bool:
		return Collection{v.Bool()}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Collection{v.Int()}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Collection{int64(v.Uint())}
	case reflect.Float32, reflect.Float64:
		return Collection{v.Float()}
	case reflect.Struct:
		return Collection{v.Interface()}
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return Collection{v.Interface()}
	default:
		return Collection{v.Interface()}
	}
}

// --- Value helpers ---

func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// ToFloat converts a value to float64.
func ToFloat(v any) float64 { return toFloat(v) }

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}

func toInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case string:
		var i int64
		fmt.Sscanf(n, "%d", &i)
		return i
	default:
		return 0
	}
}

func negate(v any) any {
	switch n := v.(type) {
	case float64:
		return -n
	case int64:
		return -n
	default:
		return v
	}
}

func valEqual(a, b any) bool {
	// Quantity comparison
	qa, ua, isQA := toQuantity(a)
	qb, ub, isQB := toQuantity(b)
	if isQA && isQB {
		if ua != "" && ub != "" && ua != ub {
			return false // different units
		}
		return qa == qb
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func collEqual(a, b Collection) bool {
	if len(a) == 0 || len(b) == 0 {
		return false // empty = anything is empty (null propagation)
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !valEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func compareValues(a, b any) int {
	// Quantity comparison
	qa, ua, isQA := toQuantity(a)
	qb, ub, isQB := toQuantity(b)
	if isQA && isQB && (ua == ub || ua == "" || ub == "") {
		if qa < qb {
			return -1
		}
		if qa > qb {
			return 1
		}
		return 0
	}

	sa := toString(a)
	sb := toString(b)

	// Try date comparison (dates are strings that sort lexicographically)
	if isDateLike(sa) && isDateLike(sb) {
		if sa < sb {
			return -1
		}
		if sa > sb {
			return 1
		}
		return 0
	}

	// Try numeric comparison
	fa := toFloat(a)
	fb := toFloat(b)
	if fa < fb {
		return -1
	}
	if fa > fb {
		return 1
	}

	// String comparison
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}

func isDateLike(s string) bool {
	// FHIR dates start with 4 digits
	return len(s) >= 4 && s[0] >= '0' && s[0] <= '9' && s[1] >= '0' && s[1] <= '9' &&
		s[2] >= '0' && s[2] <= '9' && s[3] >= '0' && s[3] <= '9' &&
		(len(s) == 4 || s[4] == '-')
}

func isType(v any, typeName string) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	goTypeName := rv.Type().Name()

	// Direct match
	if goTypeName == typeName {
		return true
	}

	// FHIR type aliases
	switch typeName {
	case "String", "string":
		return rv.Kind() == reflect.String
	case "Integer", "integer":
		return rv.Kind() == reflect.Int || rv.Kind() == reflect.Int32 || rv.Kind() == reflect.Int64
	case "Decimal", "decimal":
		return rv.Kind() == reflect.Float64 || goTypeName == "Decimal"
	case "Boolean", "boolean":
		return rv.Kind() == reflect.Bool
	case "Quantity":
		return goTypeName == "Quantity"
	case "CodeableConcept":
		return goTypeName == "CodeableConcept"
	case "Coding":
		return goTypeName == "Coding"
	case "Reference":
		return goTypeName == "Reference"
	case "Period":
		return goTypeName == "Period"
	case "Range":
		return goTypeName == "Range"
	}

	return false
}
