package chunk

import (
	"path/filepath"

	tree_sitter_zig "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_csharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_scala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type LanguageConfig struct {
	Name          string
	Extensions    []string
	Language      *sitter.Language
	TopLevelNodes []string
	SplitNodes    []string
}

var Languages = map[string]LanguageConfig{
	"rust": {
		Name:       "rust",
		Extensions: []string{".rs"},
		Language:   sitter.NewLanguage(tree_sitter_rust.Language()),
		TopLevelNodes: []string{
			"function_item", "struct_item", "enum_item", "impl_item",
			"trait_item", "type_item", "const_item", "static_item",
			"macro_definition", "mod_item",
		},
		SplitNodes: []string{"impl_item", "trait_item", "mod_item"},
	},
	"javascript": {
		Name:       "javascript",
		Extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
		Language:   sitter.NewLanguage(tree_sitter_javascript.Language()),
		TopLevelNodes: []string{
			"function_declaration", "generator_function_declaration",
			"class_declaration", "variable_declaration", "lexical_declaration",
			"export_statement",
		},
		SplitNodes: []string{"class_declaration"},
	},
	"typescript": {
		Name:       "typescript",
		Extensions: []string{".ts", ".mts", ".cts"},
		Language:   sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript()),
		TopLevelNodes: []string{
			"function_declaration", "generator_function_declaration",
			"class_declaration", "abstract_class_declaration",
			"interface_declaration", "type_alias_declaration",
			"enum_declaration", "variable_declaration", "lexical_declaration",
			"export_statement",
		},
		SplitNodes: []string{"class_declaration", "abstract_class_declaration", "interface_declaration"},
	},
	"tsx": {
		Name:       "tsx",
		Extensions: []string{".tsx"},
		Language:   sitter.NewLanguage(tree_sitter_typescript.LanguageTSX()),
		TopLevelNodes: []string{
			"function_declaration", "generator_function_declaration",
			"class_declaration", "abstract_class_declaration",
			"interface_declaration", "type_alias_declaration",
			"enum_declaration", "variable_declaration", "lexical_declaration",
			"export_statement",
		},
		SplitNodes: []string{"class_declaration", "abstract_class_declaration", "interface_declaration"},
	},
	"c": {
		Name:       "c",
		Extensions: []string{".c", ".h"},
		Language:   sitter.NewLanguage(tree_sitter_c.Language()),
		TopLevelNodes: []string{
			"function_definition", "declaration", "type_definition",
			"enum_specifier", "struct_specifier", "preproc_def", "preproc_function_def",
		},
		SplitNodes: []string{},
	},
	"cpp": {
		Name:       "cpp",
		Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		Language:   sitter.NewLanguage(tree_sitter_cpp.Language()),
		TopLevelNodes: []string{
			"function_definition", "class_specifier", "struct_specifier",
			"enum_specifier", "namespace_definition", "template_declaration",
			"declaration",
		},
		SplitNodes: []string{"class_specifier", "struct_specifier", "namespace_definition"},
	},
	"python": {
		Name:       "python",
		Extensions: []string{".py", ".pyi"},
		Language:   sitter.NewLanguage(tree_sitter_python.Language()),
		TopLevelNodes: []string{
			"function_definition", "class_definition", "decorated_definition",
		},
		SplitNodes: []string{"class_definition"},
	},
	"go": {
		Name:       "go",
		Extensions: []string{".go"},
		Language:   sitter.NewLanguage(tree_sitter_go.Language()),
		TopLevelNodes: []string{
			"function_declaration", "method_declaration",
			"type_declaration", "const_declaration", "var_declaration",
		},
		SplitNodes: []string{},
	},
	"zig": {
		Name:       "zig",
		Extensions: []string{".zig"},
		Language:   sitter.NewLanguage(tree_sitter_zig.Language()),
		TopLevelNodes: []string{
			"function_declaration", "variable_declaration", "test_declaration",
		},
		SplitNodes: []string{},
	},
	"java": {
		Name:       "java",
		Extensions: []string{".java"},
		Language:   sitter.NewLanguage(tree_sitter_java.Language()),
		TopLevelNodes: []string{
			"class_declaration", "interface_declaration",
			"enum_declaration", "record_declaration",
		},
		SplitNodes: []string{"class_declaration", "interface_declaration", "enum_declaration"},
	},
	"scala": {
		Name:       "scala",
		Extensions: []string{".scala", ".sc"},
		Language:   sitter.NewLanguage(tree_sitter_scala.Language()),
		TopLevelNodes: []string{
			"class_definition", "object_definition", "trait_definition",
			"function_definition", "val_definition",
		},
		SplitNodes: []string{"class_definition", "object_definition", "trait_definition"},
	},
	"php": {
		Name:       "php",
		Extensions: []string{".php"},
		Language:   sitter.NewLanguage(tree_sitter_php.LanguagePHPOnly()),
		TopLevelNodes: []string{
			"class_declaration", "interface_declaration", "trait_declaration",
			"function_definition", "enum_declaration",
		},
		SplitNodes: []string{"class_declaration", "interface_declaration", "trait_declaration"},
	},
	"csharp": {
		Name:       "c_sharp",
		Extensions: []string{".cs"},
		Language:   sitter.NewLanguage(tree_sitter_csharp.Language()),
		TopLevelNodes: []string{
			"class_declaration", "interface_declaration", "struct_declaration",
			"enum_declaration", "record_declaration", "method_declaration",
			"namespace_declaration",
		},
		SplitNodes: []string{"class_declaration", "interface_declaration", "struct_declaration", "namespace_declaration"},
	},
	"ruby": {
		Name:       "ruby",
		Extensions: []string{".rb"},
		Language:   sitter.NewLanguage(tree_sitter_ruby.Language()),
		TopLevelNodes: []string{
			"module", "class", "method", "singleton_method", "assignment",
		},
		SplitNodes: []string{"module", "class"},
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
