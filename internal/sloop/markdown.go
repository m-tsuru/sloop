package sloop

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type EditableDocument struct {
	ID      string   `yaml:"id"`
	Title   string   `yaml:"title"`
	Status  string   `yaml:"status"`
	Parents []string `yaml:"parents"`
	Body    string   `yaml:"-"`
}

var forbiddenEditableFields = map[string]bool{
	"project": true, "revision": true, "revision-hash": true, "log": true,
	"git-relations": true, "review-results": true, "agent-runs": true,
}

func ParseEditable(data []byte) (EditableDocument, error) {
	text := normalizeLF(string(data))
	if !strings.HasPrefix(text, "---\n") {
		return EditableDocument{}, errors.New("editable Markdown must begin with YAML front matter")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return EditableDocument{}, errors.New("editable Markdown has unterminated YAML front matter")
	}
	end += 4
	frontMatter := text[4:end]
	body := text[end+5:]

	var fields map[string]any
	if err := yaml.Unmarshal([]byte(frontMatter), &fields); err != nil {
		return EditableDocument{}, fmt.Errorf("parse YAML front matter: %w", err)
	}
	for field := range fields {
		if forbiddenEditableFields[field] {
			return EditableDocument{}, fmt.Errorf("front matter field %q is read-only", field)
		}
	}
	var doc EditableDocument
	if err := yaml.Unmarshal([]byte(frontMatter), &doc); err != nil {
		return EditableDocument{}, fmt.Errorf("parse editable front matter: %w", err)
	}
	if strings.TrimSpace(doc.ID) == "" {
		return EditableDocument{}, errors.New("front matter field \"id\" is required")
	}
	status, err := ParseStatus(doc.Status)
	if err != nil {
		return EditableDocument{}, err
	}
	doc.Status = string(status)
	doc.Body = normalizeBody(body)
	if doc.Parents == nil {
		doc.Parents = []string{}
	}
	return doc, nil
}

func RenderEditable(spec Specification) string {
	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString("id: " + strconv.Quote(spec.ID) + "\n")
	out.WriteString("title: " + strconv.Quote(spec.Title) + "\n")
	out.WriteString("status: " + string(spec.Status) + "\n")
	if len(spec.Parents) == 0 {
		out.WriteString("parents: []\n")
	} else {
		out.WriteString("parents:\n")
		for _, parent := range spec.Parents {
			out.WriteString("  - " + strconv.Quote(parent) + "\n")
		}
	}
	out.WriteString("---\n\n")
	out.WriteString(strings.TrimPrefix(normalizeBody(spec.Body), "\n"))
	return out.String()
}

func normalizeLF(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
}

func normalizeBody(value string) string {
	value = normalizeLF(value)
	value = strings.TrimLeft(value, "\n")
	if value != "" && !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	return value
}

var sectionHeading = regexp.MustCompile(`^#{1,6}[ \t]+.*?[ \t]+\{#([A-Za-z0-9_-]+)\}[ \t]*$`)

// Sections returns section bodies keyed by explicit Markdown section ID.
func Sections(markdown string) map[string]string {
	type position struct {
		id    string
		start int
		end   int
	}
	var positions []position
	scanner := bufio.NewScanner(strings.NewReader(normalizeLF(markdown)))
	offset := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineLength := len(line) + 1
		if match := sectionHeading.FindStringSubmatch(line); match != nil {
			if len(positions) > 0 {
				positions[len(positions)-1].end = offset
			}
			positions = append(positions, position{id: match[1], start: offset + lineLength})
		}
		offset += lineLength
	}
	if len(positions) > 0 {
		positions[len(positions)-1].end = len(normalizeLF(markdown))
	}
	result := make(map[string]string, len(positions))
	content := normalizeLF(markdown)
	for _, pos := range positions {
		end := pos.end
		start := pos.start
		if start > len(content) {
			start = len(content)
		}
		if end < start {
			end = start
		}
		result[pos.id] = strings.TrimSpace(content[start:end])
	}
	return result
}

func EqualMeaning(a Specification, doc EditableDocument) bool {
	if a.Title != doc.Title || normalizeBody(a.Body) != normalizeBody(doc.Body) || len(a.Parents) != len(doc.Parents) {
		return false
	}
	for i := range a.Parents {
		if a.Parents[i] != doc.Parents[i] {
			return false
		}
	}
	return true
}
