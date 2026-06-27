package main

import (
	"go/ast"
	"go/token"
	"strings"
)

func parseDirective(line string) (Directive, bool) {
	text := strings.TrimSpace(line)
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "routegen:") {
		return Directive{}, false
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "routegen:"))
	var directive Directive
	for _, field := range strings.Fields(text) {
		switch {
		case field == "auth":
			directive.Auth = true
		case field == "user_id":
			directive.UserID = true
		case field == "sse":
			directive.SSE = true
		case strings.HasPrefix(field, "input="):
			directive.Input = strings.TrimPrefix(field, "input=")
		case strings.HasPrefix(field, "output="):
			directive.Output = strings.TrimPrefix(field, "output=")
		}
	}
	return directive, true
}

func parseAtDirective(line string) (key string, value string, ok bool) {
	text := cleanCommentText(line)
	if !strings.HasPrefix(text, "@") {
		return "", "", false
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "@"))
	if text == "" {
		return "", "", false
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", "", false
	}
	key = parts[0]
	value = strings.TrimSpace(strings.TrimPrefix(text, key))
	return key, value, true
}

func parseDirectiveGroup(group *ast.CommentGroup) (Directive, []string, bool) {
	if group == nil {
		return Directive{}, nil, false
	}
	commentLines := logicCommentLinesFromGroup(group)

	for i := len(group.List) - 1; i >= 0; i-- {
		if directive, ok := parseDirective(group.List[i].Text); ok {
			return directive, commentLines, true
		}
	}

	var directive Directive
	enabled := false
	for _, item := range group.List {
		key, value, ok := parseAtDirective(item.Text)
		if !ok {
			continue
		}
		switch key {
		case "routegen":
			enabled = true
		case "auth":
			directive.Auth = true
		case "userID", "user_id":
			directive.UserID = true
		case "sse":
			directive.SSE = true
		case "input":
			directive.Input = value
		case "output":
			directive.Output = value
		}
	}
	if !enabled {
		return Directive{}, nil, false
	}
	return directive, commentLines, true
}

func directiveForCall(fset *token.FileSet, file *ast.File, call ast.Node) (Directive, []string, bool) {
	callLine := fset.Position(call.Pos()).Line
	for _, group := range file.Comments {
		if fset.Position(group.End()).Line != callLine-1 {
			continue
		}
		if directive, commentLines, ok := parseDirectiveGroup(group); ok {
			return directive, commentLines, true
		}
	}
	return Directive{}, nil, false
}

func logicCommentLinesFromGroup(group *ast.CommentGroup) []string {
	if group == nil {
		return nil
	}
	var lines []string
	for _, item := range group.List {
		text := cleanCommentText(item.Text)
		if text == "" {
			continue
		}
		if _, ok := parseDirective(item.Text); ok {
			continue
		}
		if isRoutegenAtDirective(item.Text) {
			continue
		}
		lines = append(lines, text)
	}
	return lines
}

func isRoutegenAtDirective(line string) bool {
	key, _, ok := parseAtDirective(line)
	if !ok {
		return false
	}
	switch key {
	case "routegen", "auth", "userID", "user_id", "sse", "input", "output":
		return true
	default:
		return strings.HasPrefix(key, "routegen.")
	}
}

func cleanCommentText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	return text
}
