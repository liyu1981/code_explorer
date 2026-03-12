# Plan: Enhance Codemogger with Tree-sitter Parsing

This plan outlines the steps to replace the current naive line-scanning chunker in `pkg/codemogger/chunk` with a robust Tree-sitter-based parser in Go, mirroring the full set of 14 languages and logic found in the reference TypeScript implementation.

## 1. Research & Dependency Management
- [ ] Research `github.com/tree-sitter/go-tree-sitter` (v0.25+) integration with language-specific grammars in Go.
- [ ] Add `github.com/tree-sitter/go-tree-sitter` to `go.mod`.
- [ ] Add all 14 language grammars as dependencies (or link via CGO):
    - **Systems:** Go, Rust, Zig, C, C++
    - **Web/Scripting:** JavaScript, TypeScript, TSX, Python, PHP, Ruby
    - **Enterprise/JVM:** Java, Scala, C#
- [ ] Verify CGO build compatibility for each grammar package in the target environment.

## 2. Infrastructure & Configuration
- [ ] Update `LanguageConfig` in `pkg/codemogger/chunk/chunker.go` to support Tree-sitter.
    - Store the `*sitter.Language` or a pointer to the grammar's `GetLanguage()` function.
    - Port full `topLevelNodes` and `splitNodes` configurations for all 14 languages from `languages.ts`.
- [ ] Port the `nodeKind` normalization mapping from `treesitter.ts` to Go (e.g., mapping `type_item`, `interface_declaration` etc. to common kinds like `type` or `interface`).

## 3. Core Parser Implementation
- [ ] Implement name extraction (`extractName`) in Go, porting all special cases from the TS reference:
    - **Go:** Method receivers (`(u *User) GetName` -> `User.GetName`), type/const/var specs.
    - **JS/TS:** Export unwrapping and lexical declarations.
    - **Python:** Decorated definitions (`@deco def func`).
    - **C/C++:** Template declarations and nested declarators.
    - **Ruby:** Singleton methods (`self.method`) and assignments.
    - **Zig:** String-based test declarations and identifier-based variable declarations.
    - **Scala:** Pattern-based `val_definition`.
- [ ] Implement signature extraction (`extractSignature`) to capture the first line of a node as its display signature.
- [ ] Implement the `ChunkFile` function using the Tree-sitter AST:
    - Handle unwrapping of containers (`export_statement`, `decorated_definition`, `template_declaration`).
    - Implement the logic to split "large" nodes (e.g., classes/impls/modules > 150 lines) into sub-items.
    - Use `bodyWrappers` (`class_body`, `declaration_list`, `field_declaration_list`, `block` for Python classes) to navigate and extract sub-chunks when splitting.

## 4. Refactoring & Integration
- [ ] Replace the existing naive `ChunkFile` and `DetectLanguage` logic in `pkg/codemogger/chunk/chunker.go`.
- [ ] Maintain the `CodeChunk` struct compatibility to minimize changes in `pkg/codemogger/index.go`.

## 5. Testing & Validation
- [ ] Update `pkg/codemogger/chunk/chunker_test.go` with test cases for all 14 languages.
- [ ] **Specific Validation Points:**
    - Go: Correct `Receiver.Method` naming.
    - TS: Correct handling of `export class`.
    - Python: Correct naming for decorated methods.
    - Ruby: Correct naming for `self.method`.
    - Zig: Correct handling of `test "name"`.
    - Splitting: Verify a 200-line class is split into individual method chunks.
- [ ] Format all Go code (`go fmt ./...`).

## 6. Optimization
- [ ] Investigate thread-safety of the Tree-sitter `Parser` and implement a pool if necessary for concurrent indexing.
- [ ] Ensure proper memory management by calling `Close()` or `Delete()` on AST trees.
