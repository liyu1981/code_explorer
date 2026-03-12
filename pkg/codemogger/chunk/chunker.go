package chunk

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const MAX_CHUNK_LINES = 150

type CodeChunk struct {
	ChunkKey  string
	FilePath  string
	Language  string
	Kind      string
	Name      string
	Signature string
	Snippet   string
	StartLine int
	EndLine   int
	FileHash  string
}

func ChunkFile(filePath string, content string, fileHash string, config *LanguageConfig) []CodeChunk {
	parser := sitter.NewParser()
	if err := parser.SetLanguage(config.Language); err != nil {
		return nil
	}

	tree := parser.Parse([]byte(content), nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()

	sourceLines := strings.Split(content, "\n")
	var chunks []CodeChunk

	topLevelSet := make(map[string]bool)
	for _, nodeType := range config.TopLevelNodes {
		topLevelSet[nodeType] = true
	}

	splitSet := make(map[string]bool)
	for _, nodeType := range config.SplitNodes {
		splitSet[nodeType] = true
	}

	makeChunk := func(node *sitter.Node, kind string) CodeChunk {
		startLine := int(node.StartPosition().Row) + 1
		endLine := int(node.EndPosition().Row) + 1
		name := extractName(node, content)
		signature := extractSignature(node, sourceLines)
		snippet := content[node.StartByte():node.EndByte()]

		return CodeChunk{
			ChunkKey:  fmt.Sprintf("%s:%d:%d", filePath, startLine, endLine),
			FilePath:  filePath,
			Language:  config.Name,
			Kind:      kind,
			Name:      name,
			Signature: signature,
			Snippet:   snippet,
			StartLine: startLine,
			EndLine:   endLine,
			FileHash:  fileHash,
		}
	}

	var splitLargeNode func(node *sitter.Node, outerNode *sitter.Node)
	splitLargeNode = func(node *sitter.Node, outerNode *sitter.Node) {
		hasSubItems := false

		isSubItem := func(nodeKind string) bool {
			return topLevelSet[nodeKind] ||
				strings.Contains(nodeKind, "function") ||
				strings.Contains(nodeKind, "method") ||
				strings.Contains(nodeKind, "constructor")
		}

		bodyWrappers := map[string]bool{
			"class_body":             true,
			"declaration_list":       true,
			"field_declaration_list": true,
			"body_statement":         true,
			"block":                  true,
		}

		for i := uint(0); i < node.ChildCount(); i++ {
			sub := node.Child(i)
			if isSubItem(sub.Kind()) {
				chunks = append(chunks, makeChunk(sub, nodeKind(sub.Kind())))
				hasSubItems = true
			} else if bodyWrappers[sub.Kind()] {
				for j := uint(0); j < sub.ChildCount(); j++ {
					inner := sub.Child(j)
					if isSubItem(inner.Kind()) {
						chunks = append(chunks, makeChunk(inner, nodeKind(inner.Kind())))
						hasSubItems = true
					}
				}
			}
		}

		if !hasSubItems {
			chunks = append(chunks, makeChunk(outerNode, nodeKind(node.Kind())))
		}
	}

	var processNode func(node *sitter.Node)
	processNode = func(node *sitter.Node) {
		if node.Kind() == "export_statement" {
			inner := unwrapExport(node)
			if inner != nil && topLevelSet[inner.Kind()] {
				kind := nodeKind(inner.Kind())
				lineCount := int(node.EndPosition().Row - node.StartPosition().Row + 1)
				if lineCount <= MAX_CHUNK_LINES || !splitSet[inner.Kind()] {
					chunks = append(chunks, makeChunk(node, kind))
				} else {
					splitLargeNode(inner, node)
				}
				return
			}
			if inner != nil && (strings.Contains(inner.Kind(), "function") || strings.Contains(inner.Kind(), "class")) {
				chunks = append(chunks, makeChunk(node, nodeKind(inner.Kind())))
			}
			return
		}

		if node.Kind() == "decorated_definition" {
			inner := node.ChildByFieldName("definition")
			if inner != nil {
				kind := nodeKind(inner.Kind())
				lineCount := int(node.EndPosition().Row - node.StartPosition().Row + 1)
				if lineCount <= MAX_CHUNK_LINES || !splitSet[inner.Kind()] {
					chunks = append(chunks, makeChunk(node, kind))
				} else {
					splitLargeNode(inner, node)
				}
				return
			}
		}

		if node.Kind() == "template_declaration" {
			var inner *sitter.Node
			for i := uint(0); i < node.NamedChildCount(); i++ {
				child := node.NamedChild(i)
				if child.Kind() != "template_parameter_list" {
					inner = child
					break
				}
			}
			if inner != nil {
				kind := nodeKind(inner.Kind())
				lineCount := int(node.EndPosition().Row - node.StartPosition().Row + 1)
				if lineCount <= MAX_CHUNK_LINES || !splitSet[inner.Kind()] {
					chunks = append(chunks, makeChunk(node, kind))
				} else {
					splitLargeNode(inner, node)
				}
				return
			}
		}

		if !topLevelSet[node.Kind()] {
			return
		}

		lineCount := int(node.EndPosition().Row - node.StartPosition().Row + 1)
		kind := nodeKind(node.Kind())

		if lineCount <= MAX_CHUNK_LINES || !splitSet[node.Kind()] {
			chunks = append(chunks, makeChunk(node, kind))
			return
		}

		splitLargeNode(node, node)
	}

	root := tree.RootNode()
	for i := uint(0); i < root.ChildCount(); i++ {
		processNode(root.Child(i))
	}

	return chunks
}

func extractName(node *sitter.Node, content string) string {
	if node.Kind() == "export_statement" {
		inner := unwrapExport(node)
		if inner != nil {
			return extractName(inner, content)
		}
		return ""
	}
	if node.Kind() == "decorated_definition" {
		inner := node.ChildByFieldName("definition")
		if inner != nil {
			return extractName(inner, content)
		}
		return ""
	}
	if node.Kind() == "template_declaration" {
		var inner *sitter.Node
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() != "template_parameter_list" {
				inner = child
				break
			}
		}
		if inner != nil {
			return extractName(inner, content)
		}
		return ""
	}
	if node.Kind() == "singleton_method" {
		obj := node.ChildByFieldName("object")
		nameNode := node.ChildByFieldName("name")
		if obj != nil && nameNode != nil {
			return fmt.Sprintf("%s.%s", content[obj.StartByte():obj.EndByte()], content[nameNode.StartByte():nameNode.EndByte()])
		}
		if nameNode != nil {
			return content[nameNode.StartByte():nameNode.EndByte()]
		}
	}
	if node.Kind() == "assignment" {
		if node.NamedChildCount() > 0 {
			left := node.NamedChild(0)
			return content[left.StartByte():left.EndByte()]
		}
		return ""
	}
	if node.Kind() == "function_definition" {
		declarator := node.ChildByFieldName("declarator")
		if declarator != nil && declarator.Kind() == "function_declarator" {
			fnName := declarator.ChildByFieldName("declarator")
			if fnName != nil {
				return content[fnName.StartByte():fnName.EndByte()]
			}
		}
	}
	if node.Kind() == "type_definition" {
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == "type_identifier" {
				return content[child.StartByte():child.EndByte()]
			}
		}
	}
	if node.Kind() == "method_declaration" {
		nameNode := node.ChildByFieldName("name")
		receiver := node.ChildByFieldName("receiver")
		if nameNode != nil && receiver != nil {
			if receiver.NamedChildCount() > 0 {
				param := receiver.NamedChild(0)
				// Look for type child in parameter_declaration
				var typeNode *sitter.Node
				for j := uint(0); j < param.NamedChildCount(); j++ {
					child := param.NamedChild(j)
					if strings.Contains(child.Kind(), "type") || child.Kind() == "identifier" && j > 0 {
						typeNode = child
					}
				}
				if typeNode != nil {
					typeName := content[typeNode.StartByte():typeNode.EndByte()]
					typeName = strings.TrimPrefix(typeName, "*")
					return fmt.Sprintf("%s.%s", typeName, content[nameNode.StartByte():nameNode.EndByte()])
				}
			}
		}
		if nameNode != nil {
			return content[nameNode.StartByte():nameNode.EndByte()]
		}
	}
	if node.Kind() == "type_declaration" {
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == "type_spec" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					return content[nameNode.StartByte():nameNode.EndByte()]
				}
			}
		}
	}
	if node.Kind() == "const_declaration" || node.Kind() == "var_declaration" {
		specType := "const_spec"
		if node.Kind() == "var_declaration" {
			specType = "var_spec"
		}
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == specType {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					return content[nameNode.StartByte():nameNode.EndByte()]
				}
			}
		}
	}
	if node.Kind() == "val_definition" {
		pattern := node.ChildByFieldName("pattern")
		if pattern != nil {
			return content[pattern.StartByte():pattern.EndByte()]
		}
	}
	if node.Kind() == "variable_declaration" {
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == "identifier" {
				return content[child.StartByte():child.EndByte()]
			}
		}
	}
	if node.Kind() == "test_declaration" {
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == "string" || child.Kind() == "string_literal" {
				return strings.Trim(content[child.StartByte():child.EndByte()], "\"")
			}
		}
	}

	for _, field := range []string{"name", "identifier", "type_identifier"} {
		child := node.ChildByFieldName(field)
		if child != nil {
			return content[child.StartByte():child.EndByte()]
		}
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		traitNode := node.ChildByFieldName("trait")
		if traitNode != nil {
			return fmt.Sprintf("%s for %s", content[traitNode.StartByte():traitNode.EndByte()], content[typeNode.StartByte():typeNode.EndByte()])
		}
		return content[typeNode.StartByte():typeNode.EndByte()]
	}

	if node.Kind() == "lexical_declaration" {
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() == "variable_declarator" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					return content[nameNode.StartByte():nameNode.EndByte()]
				}
			}
		}
	}

	return ""
}

func unwrapExport(node *sitter.Node) *sitter.Node {
	if node.Kind() != "export_statement" {
		return nil
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() != "decorator" && child.Kind() != "comment" {
			return child
		}
	}
	return nil
}

func extractSignature(node *sitter.Node, sourceLines []string) string {
	startLine := int(node.StartPosition().Row)
	if startLine < len(sourceLines) {
		return strings.TrimSpace(sourceLines[startLine])
	}
	return ""
}

func nodeKind(nodeKind string) string {
	if strings.Contains(nodeKind, "function") || nodeKind == "function_item" {
		return "function"
	}
	if strings.Contains(nodeKind, "struct") {
		return "struct"
	}
	if strings.Contains(nodeKind, "enum") {
		return "enum"
	}
	if strings.Contains(nodeKind, "impl") {
		return "impl"
	}
	if strings.Contains(nodeKind, "trait") {
		return "trait"
	}
	if nodeKind == "type_item" || nodeKind == "type_alias_declaration" || nodeKind == "type_definition" || nodeKind == "type_declaration" {
		return "type"
	}
	if strings.Contains(nodeKind, "const") {
		return "const"
	}
	if strings.Contains(nodeKind, "static") {
		return "static"
	}
	if strings.Contains(nodeKind, "macro") || nodeKind == "preproc_def" || nodeKind == "preproc_function_def" {
		return "macro"
	}
	if nodeKind == "namespace_definition" || nodeKind == "namespace_declaration" {
		return "namespace"
	}
	if nodeKind == "template_declaration" {
		return "template"
	}
	if strings.Contains(nodeKind, "mod") {
		return "module"
	}
	if strings.Contains(nodeKind, "class") {
		return "class"
	}
	if nodeKind == "method_declaration" || strings.Contains(nodeKind, "method") {
		if strings.Contains(nodeKind, "definition") {
			return "function"
		}
		return "method"
	}
	if strings.Contains(nodeKind, "interface") {
		return "interface"
	}
	if nodeKind == "variable_declaration" || nodeKind == "lexical_declaration" || nodeKind == "var_declaration" || nodeKind == "val_definition" || nodeKind == "assignment" {
		return "variable"
	}
	if nodeKind == "declaration" {
		return "declaration"
	}
	if nodeKind == "decorated_definition" {
		return "function"
	}
	if nodeKind == "test_declaration" {
		return "test"
	}
	if nodeKind == "object_definition" {
		return "object"
	}
	if nodeKind == "record_declaration" {
		return "record"
	}
	if nodeKind == "constructor_declaration" {
		return "constructor"
	}
	return nodeKind
}
