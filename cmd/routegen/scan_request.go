package main

import (
	"go/ast"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func scanRequestDTOs(root string) (map[string]RequestDTO, error) {
	requestDir := filepath.Join(root, "internal", "transport", "http", "request")
	structs := map[string]requestStruct{}
	if err := scanGoFiles(requestDir, func(path string, file *ast.File) {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				structs[typeSpec.Name.Name] = requestStruct{Fields: requestFields(structType)}
			}
		}
	}); err != nil {
		return nil, err
	}

	out := map[string]RequestDTO{}
	for name := range structs {
		out[name] = RequestDTO{
			HasJSONBody:      requestStructHasJSONBody(name, structs, map[string]bool{}),
			HasMultipartBody: requestStructHasMultipartBody(name, structs, map[string]bool{}),
		}
	}
	return out, nil
}

func requestFields(structType *ast.StructType) []requestField {
	if structType == nil || structType.Fields == nil {
		return nil
	}
	fields := make([]requestField, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		item := requestField{}
		if field.Tag != nil {
			item.JSONTag = tagValue(field.Tag.Value, "json")
			item.URITag = tagValue(field.Tag.Value, "uri")
			item.FormTag = tagValue(field.Tag.Value, "form")
		}
		if len(field.Names) == 0 {
			item.EmbeddedType = requestEmbeddedTypeName(field.Type)
		}
		item.HasMultipartFile = tagName(item.FormTag) != "" && requestExprIsMultipartFileHeader(field.Type)
		fields = append(fields, item)
	}
	return fields
}

func requestStructHasJSONBody(name string, structs map[string]requestStruct, visiting map[string]bool) bool {
	if visiting[name] {
		return false
	}
	info, ok := structs[name]
	if !ok {
		return false
	}
	visiting[name] = true
	defer delete(visiting, name)

	for _, field := range info.Fields {
		if tagName(field.JSONTag) != "" && tagName(field.URITag) == "" {
			return true
		}
		if field.EmbeddedType != "" && requestStructHasJSONBody(field.EmbeddedType, structs, visiting) {
			return true
		}
	}
	return false
}

func requestStructHasMultipartBody(name string, structs map[string]requestStruct, visiting map[string]bool) bool {
	if visiting[name] {
		return false
	}
	info, ok := structs[name]
	if !ok {
		return false
	}
	visiting[name] = true
	defer delete(visiting, name)

	for _, field := range info.Fields {
		if field.HasMultipartFile {
			return true
		}
		if field.EmbeddedType != "" && requestStructHasMultipartBody(field.EmbeddedType, structs, visiting) {
			return true
		}
	}
	return false
}

func requestEmbeddedTypeName(expr ast.Expr) string {
	switch item := expr.(type) {
	case *ast.Ident:
		return item.Name
	case *ast.StarExpr:
		return requestEmbeddedTypeName(item.X)
	case *ast.SelectorExpr:
		return item.Sel.Name
	default:
		return ""
	}
}

func requestExprIsMultipartFileHeader(expr ast.Expr) bool {
	switch item := expr.(type) {
	case *ast.SelectorExpr:
		return item.Sel.Name == "FileHeader"
	case *ast.StarExpr:
		return requestExprIsMultipartFileHeader(item.X)
	case *ast.ArrayType:
		return requestExprIsMultipartFileHeader(item.Elt)
	default:
		return false
	}
}

func tagValue(raw string, key string) string {
	if raw == "" {
		return ""
	}
	unquoted, err := strconv.Unquote(raw)
	if err != nil {
		return ""
	}
	return reflect.StructTag(unquoted).Get(key)
}

func tagName(tag string) string {
	if tag == "" {
		return ""
	}
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return ""
	}
	return name
}
