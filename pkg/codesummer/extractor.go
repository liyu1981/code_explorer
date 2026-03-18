package codesummer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct {
	parser *sitter.Parser
}

func NewExtractor() *Extractor {
	return &Extractor{
		parser: sitter.NewParser(),
	}
}

func (e *Extractor) ExtractDefinitions(filePath string, content string, language string) ([]Definition, error) {
	langConfig := chunk.Languages[language]
	if langConfig.Language == nil {
		return nil, nil
	}

	if err := e.parser.SetLanguage(langConfig.Language); err != nil {
		return nil, err
	}

	tree := e.parser.Parse([]byte(content), nil)
	if tree == nil {
		return nil, nil
	}
	defer tree.Close()

	var definitions []Definition
	topLevelSet := make(map[string]bool)
	for _, nodeType := range langConfig.TopLevelNodes {
		topLevelSet[nodeType] = true
	}

	sourceLines := strings.Split(content, "\n")

	var processNode func(node *sitter.Node)
	processNode = func(node *sitter.Node) {
		if !topLevelSet[node.Kind()] {
			return
		}

		name := extractName(node, content)
		signature := extractSignature(node, sourceLines)
		kind := nodeKind(node.Kind())

		definitions = append(definitions, Definition{
			Kind:      kind,
			Name:      name,
			Signature: signature,
		})
	}

	root := tree.RootNode()
	for i := uint(0); i < root.ChildCount(); i++ {
		processNode(root.Child(i))
	}

	return definitions, nil
}

func (e *Extractor) ReadFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (e *Extractor) ComputeFileHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := hashContent(string(content))
	return hash, nil
}

func extractName(node *sitter.Node, content string) string {
	for _, field := range []string{"name", "identifier", "type_identifier"} {
		child := node.ChildByFieldName(field)
		if child != nil {
			return content[child.StartByte():child.EndByte()]
		}
	}

	if node.Kind() == "method_declaration" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			return content[nameNode.StartByte():nameNode.EndByte()]
		}
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

	return ""
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
	if strings.Contains(nodeKind, "class") {
		return "class"
	}
	if strings.Contains(nodeKind, "interface") {
		return "interface"
	}
	if strings.Contains(nodeKind, "module") || strings.Contains(nodeKind, "mod") {
		return "module"
	}
	if strings.Contains(nodeKind, "macro") {
		return "macro"
	}
	if nodeKind == "variable_declaration" || nodeKind == "lexical_declaration" || nodeKind == "var_declaration" || nodeKind == "val_definition" {
		return "variable"
	}
	if nodeKind == "namespace_definition" || nodeKind == "namespace_declaration" {
		return "namespace"
	}
	return nodeKind
}

func hashContent(content string) string {
	h := uint32(2166136261)
	for _, c := range content {
		h ^= uint32(c)
		h *= 16777619
	}
	return filepath.Base("h" + string(rune(h%26+'a')) + string(rune((h/26)%26+'a')))
}
