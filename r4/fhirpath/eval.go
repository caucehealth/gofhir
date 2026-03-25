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
	"strings"
)

// Collection is the result of evaluating a FHIRPath expression.
// In FHIRPath, everything is a collection (even single values).
type Collection []any

// Expression is a compiled FHIRPath expression.
type Expression struct {
	source string
	ast    Node
}

// Compile parses a FHIRPath expression for repeated evaluation.
func Compile(expr string) (*Expression, error) {
	ast, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	return &Expression{source: expr, ast: ast}, nil
}

// Evaluate evaluates the compiled expression against a resource.
func (e *Expression) Evaluate(resource any) (Collection, error) {
	ctx := &evalContext{}
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
type evalContext struct{}

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

	case *EmptyNode:
		return nil, nil

	default:
		return nil, fmt.Errorf("fhirpath: unknown node type %T", node)
	}
}

func (ctx *evalContext) evalIdent(name string, input Collection) (Collection, error) {
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
	case "matches":
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

// --- Reflection helpers ---

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
		if tag == "" || tag == "-" {
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
	fa := toFloat(a)
	fb := toFloat(b)
	if fa < fb {
		return -1
	}
	if fa > fb {
		return 1
	}
	// Try string comparison
	sa := toString(a)
	sb := toString(b)
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
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
