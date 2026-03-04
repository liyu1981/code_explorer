package chunk

import (
	"path/filepath"
	"strconv"
	"strings"
)

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

type LanguageConfig struct {
	Name          string
	Extensions    []string
	TopLevelNodes []string
	SplitNodes    []string
}

var Languages = map[string]LanguageConfig{
	"rust": {
		Name:       "rust",
		Extensions: []string{".rs"},
		TopLevelNodes: []string{
			"function_item", "struct_item", "enum_item", "impl_item",
			"trait_item", "type_item", "const_item", "static_item",
			"macro_definition", "mod_item",
		},
		SplitNodes: []string{"impl_item", "trait_item", "mod_item"},
	},
	"go": {
		Name:       "go",
		Extensions: []string{".go"},
		TopLevelNodes: []string{
			"function_declaration", "method_declaration",
			"type_declaration", "const_declaration", "var_declaration",
		},
		SplitNodes: []string{},
	},
	"python": {
		Name:       "python",
		Extensions: []string{".py", ".pyi"},
		TopLevelNodes: []string{
			"function_definition", "class_definition", "decorated_definition",
		},
		SplitNodes: []string{"class_definition"},
	},
	"typescript": {
		Name:       "typescript",
		Extensions: []string{".ts", ".mts", ".cts"},
		TopLevelNodes: []string{
			"function_declaration", "generator_function_declaration",
			"class_declaration", "abstract_class_declaration",
			"interface_declaration", "type_alias_declaration",
			"enum_declaration", "variable_declaration", "lexical_declaration",
			"export_statement",
		},
		SplitNodes: []string{"class_declaration", "abstract_class_declaration", "interface_declaration"},
	},
	"javascript": {
		Name:       "javascript",
		Extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
		TopLevelNodes: []string{
			"function_declaration", "generator_function_declaration",
			"class_declaration", "variable_declaration", "lexical_declaration",
			"export_statement",
		},
		SplitNodes: []string{"class_declaration"},
	},
	"tsx": {
		Name:       "tsx",
		Extensions: []string{".tsx"},
		TopLevelNodes: []string{
			"function_declaration", "generator_function_declaration",
			"class_declaration", "abstract_class_declaration",
			"interface_declaration", "type_alias_declaration",
			"enum_declaration", "variable_declaration", "lexical_declaration",
			"export_statement",
		},
		SplitNodes: []string{"class_declaration", "abstract_class_declaration", "interface_declaration"},
	},
	"java": {
		Name:       "java",
		Extensions: []string{".java"},
		TopLevelNodes: []string{
			"class_declaration", "interface_declaration",
			"enum_declaration", "record_declaration",
		},
		SplitNodes: []string{"class_declaration", "interface_declaration", "enum_declaration"},
	},
	"c": {
		Name:       "c",
		Extensions: []string{".c", ".h"},
		TopLevelNodes: []string{
			"function_definition", "declaration", "type_definition",
			"enum_specifier", "struct_specifier", "preproc_def", "preproc_function_def",
		},
		SplitNodes: []string{},
	},
	"cpp": {
		Name:       "cpp",
		Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		TopLevelNodes: []string{
			"function_definition", "class_specifier", "struct_specifier",
			"enum_specifier", "namespace_definition", "template_declaration",
			"declaration",
		},
		SplitNodes: []string{"class_specifier", "struct_specifier", "namespace_definition"},
	},
}

func DetectLanguage(filePath string) *LanguageConfig {
	ext := filepath.Ext(filePath)
	for _, lang := range Languages {
		for _, e := range lang.Extensions {
			if e == ext {
				langCfg := lang
				return &langCfg
			}
		}
	}
	return nil
}

func SupportedExtensions() []string {
	var exts []string
	seen := make(map[string]bool)
	for _, lang := range Languages {
		for _, ext := range lang.Extensions {
			if !seen[ext] {
				seen[ext] = true
				exts = append(exts, ext)
			}
		}
	}
	return exts
}

func ChunkFile(filePath string, content string, fileHash string, config *LanguageConfig) []CodeChunk {
	lines := strings.Split(content, "\n")
	var chunks []CodeChunk

	topLevelKinds := []string{
		"func ", "func(", "function ", "def ", "class ", "struct ",
		"interface ", "type ", "enum ", "const ", "var ", "pub ",
		"impl ", "trait ", "mod ", "module ",
	}

	currentBlock := ""
	blockStart := 0
	blockKind := ""
	blockName := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			if currentBlock != "" {
				currentBlock += line + "\n"
			}
			continue
		}

		isTopLevel := false
		foundKind := ""
		for _, k := range topLevelKinds {
			if strings.HasPrefix(trimmed, k) || (k == "pub " && strings.HasPrefix(trimmed, "pub ")) {
				isTopLevel = true
				foundKind = k
				break
			}
		}

		if isTopLevel {
			if currentBlock != "" {
				chunks = append(chunks, CodeChunk{
					ChunkKey:  filePath + ":" + strconv.Itoa(blockStart+1) + ":" + strconv.Itoa(i),
					FilePath:  filePath,
					Language:  config.Name,
					Kind:      blockKind,
					Name:      blockName,
					Signature: strings.TrimSpace(lines[blockStart]),
					Snippet:   currentBlock,
					StartLine: blockStart + 1,
					EndLine:   i,
					FileHash:  fileHash,
				})
			}

			currentBlock = line + "\n"
			blockStart = i
			blockKind = "function"
			if strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "struct ") {
				blockKind = strings.TrimSpace(strings.Split(trimmed, " ")[0])
			} else if strings.HasPrefix(trimmed, "type ") {
				blockKind = "type"
			} else if strings.HasPrefix(trimmed, "const ") {
				blockKind = "const"
			} else if strings.HasPrefix(trimmed, "var ") {
				blockKind = "var"
			}

			// Try to extract name
			remaining := strings.TrimPrefix(trimmed, foundKind)
			remaining = strings.TrimSpace(remaining)
			if idx := strings.IndexAny(remaining, " ([:{;"); idx > 0 {
				blockName = remaining[:idx]
			} else {
				blockName = remaining
			}
		} else {
			currentBlock += line + "\n"
		}
	}

	if currentBlock != "" {
		chunks = append(chunks, CodeChunk{
			ChunkKey:  filePath + ":" + strconv.Itoa(blockStart+1) + ":" + strconv.Itoa(len(lines)),
			FilePath:  filePath,
			Language:  config.Name,
			Kind:      blockKind,
			Name:      blockName,
			Signature: strings.TrimSpace(lines[blockStart]),
			Snippet:   currentBlock,
			StartLine: blockStart + 1,
			EndLine:   len(lines),
			FileHash:  fileHash,
		})
	}

	if len(chunks) == 0 && len(lines) > 0 {
		chunks = append(chunks, CodeChunk{
			ChunkKey:  filePath + ":1:" + strconv.Itoa(len(lines)),
			FilePath:  filePath,
			Language:  config.Name,
			Kind:      "file",
			Name:      filepath.Base(filePath),
			Signature: "",
			Snippet:   content,
			StartLine: 1,
			EndLine:   len(lines),
			FileHash:  fileHash,
		})
	}

	return chunks
}
