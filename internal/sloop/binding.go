package sloop

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var featureIDPattern = regexp.MustCompile("^[a-z0-9][a-z0-9_-]*$")

func ValidateFeatureID(id string) error {
	if !featureIDPattern.MatchString(id) {
		return fmt.Errorf("invalid feature ID %q (expected [a-z0-9][a-z0-9_-]*)", id)
	}
	return nil
}

type SymbolLocator struct {
	Raw    string
	Path   string
	Owner  string
	Symbol string
}

func (l SymbolLocator) String() string {
	if l.Owner == "" {
		return l.Path + ":" + l.Symbol
	}
	return l.Path + ":" + l.Owner + ":" + l.Symbol
}

func ParseSymbolLocator(value string) (SymbolLocator, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return SymbolLocator{}, fmt.Errorf("invalid symbol locator %q (expected path:symbol or path:type:method)", value)
	}
	for _, part := range parts {
		if part == "" || strings.TrimSpace(part) != part || strings.ContainsAny(part, "\r\n\t") {
			return SymbolLocator{}, fmt.Errorf("invalid symbol locator %q", value)
		}
	}
	path := filepath.ToSlash(filepath.Clean(parts[0]))
	if path == "." || filepath.IsAbs(parts[0]) || path == ".." || strings.HasPrefix(path, "../") {
		return SymbolLocator{}, fmt.Errorf("symbol locator path must be repository-relative: %q", parts[0])
	}
	locator := SymbolLocator{Raw: value, Path: path}
	if len(parts) == 2 {
		locator.Symbol = parts[1]
	} else {
		locator.Owner = parts[1]
		locator.Symbol = parts[2]
	}
	if strings.ContainsAny(locator.Owner, "/\\") || strings.ContainsAny(locator.Symbol, "/\\") {
		return SymbolLocator{}, fmt.Errorf("invalid symbol locator %q", value)
	}
	locator.Raw = locator.String()
	return locator, nil
}

func NormalizeFeatureBindings(features FeatureBindings) (FeatureBindings, error) {
	if features == nil {
		return FeatureBindings{}, nil
	}
	normalized := make(FeatureBindings, len(features))
	for id, binding := range features {
		if err := ValidateFeatureID(id); err != nil {
			return nil, err
		}
		impls, err := normalizeLocators(binding.Impls)
		if err != nil {
			return nil, fmt.Errorf("func:%s impls: %w", id, err)
		}
		tests, err := normalizeLocators(binding.Tests)
		if err != nil {
			return nil, fmt.Errorf("func:%s tests: %w", id, err)
		}
		normalized[id] = FeatureBinding{Impls: impls, Tests: tests}
	}
	return normalized, nil
}

func normalizeLocators(values []string) ([]string, error) {
	if values == nil {
		return []string{}, nil
	}
	seen := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		locator, err := ParseSymbolLocator(value)
		if err != nil {
			return nil, err
		}
		value = locator.String()
		if !seen[value] {
			seen[value] = true
			normalized = append(normalized, value)
		}
	}
	sort.Strings(normalized)
	return normalized, nil
}

func EqualFeatureBindings(a, b FeatureBindings) bool {
	a, errA := NormalizeFeatureBindings(a)
	b, errB := NormalizeFeatureBindings(b)
	if errA != nil || errB != nil || len(a) != len(b) {
		return false
	}
	for id, left := range a {
		right, ok := b[id]
		if !ok || !equalStrings(left.Impls, right.Impls) || !equalStrings(left.Tests, right.Tests) {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func CloneFeatureBindings(features FeatureBindings) FeatureBindings {
	clone := make(FeatureBindings, len(features))
	for id, binding := range features {
		clone[id] = FeatureBinding{
			Impls: append([]string(nil), binding.Impls...),
			Tests: append([]string(nil), binding.Tests...),
		}
	}
	return clone
}

func ResolveFeatureBindings(repositoryRoot string, features FeatureBindings) ([]FeatureResolution, error) {
	normalized, err := NormalizeFeatureBindings(features)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(normalized))
	for id := range normalized {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	results := make([]FeatureResolution, 0, len(ids))
	for _, id := range ids {
		binding := normalized[id]
		result := FeatureResolution{
			ID: id, Section: "func:" + id,
			Impls: []BindingResolution{}, Tests: []BindingResolution{},
		}
		for _, value := range binding.Impls {
			resolution, err := ResolveSymbol(repositoryRoot, value)
			if err != nil {
				return nil, err
			}
			result.Impls = append(result.Impls, resolution)
		}
		for _, value := range binding.Tests {
			resolution, err := ResolveSymbol(repositoryRoot, value)
			if err != nil {
				return nil, err
			}
			result.Tests = append(result.Tests, resolution)
		}
		results = append(results, result)
	}
	return results, nil
}

func FeatureBindingsResolved(results []FeatureResolution) bool {
	for _, result := range results {
		if !result.Resolved() {
			return false
		}
	}
	return true
}

func FormatUnresolvedFeatureBindings(results []FeatureResolution, prefix, suffix string) string {
	var out strings.Builder
	if prefix != "" {
		out.WriteString(prefix)
		out.WriteString("\n")
	}
	for _, feature := range results {
		if feature.Resolved() {
			continue
		}
		fmt.Fprintf(&out, "\nfunc:%s\n", feature.ID)
		writeUnresolvedBindings(&out, "impls", feature.Impls)
		writeUnresolvedBindings(&out, "tests", feature.Tests)
	}
	if suffix != "" {
		out.WriteString("\n")
		out.WriteString(suffix)
	}
	return strings.TrimSpace(out.String())
}

func writeUnresolvedBindings(out *strings.Builder, kind string, bindings []BindingResolution) {
	first := true
	for _, binding := range bindings {
		if binding.Status == BindingResolved {
			continue
		}
		if first {
			fmt.Fprintf(out, "  %s:\n", kind)
			first = false
		}
		fmt.Fprintf(out, "    %s [%s]\n", binding.Locator, binding.Status)
	}
}

func ResolveSymbol(repositoryRoot, value string) (BindingResolution, error) {
	locator, err := ParseSymbolLocator(value)
	if err != nil {
		return BindingResolution{}, err
	}
	result := BindingResolution{Locator: locator.String(), Path: locator.Path, Symbol: locator.Symbol}
	extension := strings.ToLower(filepath.Ext(locator.Path))
	if extension != ".go" && extension != ".py" {
		result.Status = BindingUnsupported
		return result, nil
	}
	path := filepath.Join(repositoryRoot, filepath.FromSlash(locator.Path))
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		result.Status = BindingMissing
		return result, nil
	}
	if err != nil {
		return BindingResolution{}, fmt.Errorf("read symbol file %q: %w", locator.Path, err)
	}
	var matches int
	switch extension {
	case ".go":
		matches, err = resolveGoSymbol(path, data, locator)
	case ".py":
		matches, err = resolvePythonSymbol(data, locator)
	}
	if err != nil {
		return BindingResolution{}, err
	}
	switch {
	case matches == 0:
		result.Status = BindingMissing
	case matches == 1:
		result.Status = BindingResolved
	default:
		result.Status = BindingAmbiguous
	}
	return result, nil
}

func resolveGoSymbol(path string, data []byte, locator SymbolLocator) (int, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
	if err != nil {
		return 0, fmt.Errorf("parse Go symbol file %q: %w", locator.Path, err)
	}
	matches := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != locator.Symbol {
			continue
		}
		if locator.Owner == "" && function.Recv == nil {
			matches++
			continue
		}
		if locator.Owner != "" && receiverName(function.Recv) == locator.Owner {
			matches++
		}
	}
	return matches, nil
}

func receiverName(receivers *ast.FieldList) string {
	if receivers == nil || len(receivers.List) != 1 {
		return ""
	}
	return namedType(receivers.List[0].Type)
}

func namedType(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return namedType(value.X)
	case *ast.IndexExpr:
		return namedType(value.X)
	case *ast.IndexListExpr:
		return namedType(value.X)
	case *ast.ParenExpr:
		return namedType(value.X)
	default:
		return ""
	}
}

type pythonBlock struct {
	indent int
	kind   string
	name   string
}

type pythonSymbol struct {
	owner string
	name  string
}

var (
	pythonFunction = regexp.MustCompile("^(?:async[ \\t]+)?def[ \\t]+([A-Za-z_][A-Za-z0-9_]*)[ \\t]*\\(")
	pythonClass    = regexp.MustCompile("^class[ \\t]+([A-Za-z_][A-Za-z0-9_]*)\\b")
)

func resolvePythonSymbol(data []byte, locator SymbolLocator) (int, error) {
	symbols, err := pythonSymbols(string(data))
	if err != nil {
		return 0, fmt.Errorf("parse Python symbol file %q: %w", locator.Path, err)
	}
	matches := 0
	for _, symbol := range symbols {
		if symbol.owner == locator.Owner && symbol.name == locator.Symbol {
			matches++
		}
	}
	return matches, nil
}

func pythonSymbols(source string) ([]pythonSymbol, error) {
	source = strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n")
	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var blocks []pythonBlock
	var symbols []pythonSymbol
	tripleQuote := ""
	for scanner.Scan() {
		line := pythonLineOutsideTripleString(scanner.Text(), &tripleQuote)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := pythonIndent(line)
		for len(blocks) > 0 && indent <= blocks[len(blocks)-1].indent {
			blocks = blocks[:len(blocks)-1]
		}
		if match := pythonFunction.FindStringSubmatch(trimmed); match != nil {
			if len(blocks) == 0 && indent == 0 {
				symbols = append(symbols, pythonSymbol{name: match[1]})
			} else if len(blocks) > 0 && blocks[len(blocks)-1].kind == "class" {
				symbols = append(symbols, pythonSymbol{owner: blocks[len(blocks)-1].name, name: match[1]})
			}
			blocks = append(blocks, pythonBlock{indent: indent, kind: "function", name: match[1]})
			continue
		}
		if match := pythonClass.FindStringSubmatch(trimmed); match != nil {
			blocks = append(blocks, pythonBlock{indent: indent, kind: "class", name: match[1]})
			continue
		}
		if pythonCompoundStatement(trimmed) {
			blocks = append(blocks, pythonBlock{indent: indent, kind: "other"})
		}
	}
	return symbols, scanner.Err()
}

func pythonIndent(line string) int {
	indent := 0
	for _, char := range line {
		switch char {
		case ' ':
			indent++
		case '\t':
			indent += 8 - indent%8
		default:
			return indent
		}
	}
	return indent
}

func pythonCompoundStatement(line string) bool {
	if !strings.HasSuffix(strings.TrimSpace(line), ":") {
		return false
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return false
	}
	first := strings.TrimSuffix(words[0], ":")
	switch first {
	case "if", "elif", "else", "for", "while", "try", "except", "finally", "with", "match", "case":
		return true
	default:
		return false
	}
}

func pythonLineOutsideTripleString(line string, active *string) string {
	remaining := line
	var code strings.Builder
	for {
		if *active != "" {
			end := strings.Index(remaining, *active)
			if end < 0 {
				return code.String()
			}
			remaining = remaining[end+3:]
			*active = ""
			continue
		}
		single := strings.Index(remaining, "'''")
		double := strings.Index(remaining, `"""`)
		start, delimiter := earliestTripleQuote(single, double)
		if start < 0 {
			code.WriteString(remaining)
			return code.String()
		}
		code.WriteString(remaining[:start])
		remaining = remaining[start+3:]
		if end := strings.Index(remaining, delimiter); end >= 0 {
			remaining = remaining[end+3:]
			continue
		}
		*active = delimiter
		return code.String()
	}
}

func earliestTripleQuote(single, double int) (int, string) {
	switch {
	case single < 0:
		return double, `"""`
	case double < 0:
		return single, "'''"
	case single < double:
		return single, "'''"
	default:
		return double, `"""`
	}
}
