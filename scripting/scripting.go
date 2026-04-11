// Package scripting implements SentanylScript — a domain-specific scripting
// language for expressing email-automation campaign graphs.
//
// The language models the entire entity hierarchy (Story → Storyline →
// Enactment → Scene → Message/Template) with triggers, actions, conditions,
// loops, retries, badges, and conditional routing. Scripts are compiled into
// the existing Sentanyl entity structures for persistence and execution.
//
// Pipeline: Source → Lexer → Tokens → Parser → AST → Expander → Validator → Compiler → Entities
//
// The Expander phase (DSL v2) resolves high-level authoring constructs:
//   - default sender blocks → inherited from_email/from_name/reply_to
//   - links blocks → symbolic link name resolution
//   - pattern definitions → parameterized enactment/scene expansion
//   - policy definitions → parameterized trigger expansion
//   - scenes ranges → concrete scene generation from 1..N loops
//   - use statements → pattern/policy/sender application
//
// Public entry points:
//
//	ParseScript(src)               — lex + parse → AST + diagnostics
//	ExpandScript(src)              — lex + parse + expand → AST + diagnostics
//	ValidateScript(src)            — lex + parse + expand + validate → AST + symbol table + diagnostics
//	CompileScript(src, sub, cid)   — full pipeline → entities + diagnostics
package scripting

import (
	"gopkg.in/mgo.v2/bson"
)

// ParseResult bundles the output of ParseScript.
type ParseResult struct {
	AST         *ScriptAST
	Diagnostics Diagnostics
}

// ValidateResult bundles the output of ValidateScript.
type ValidateResult struct {
	AST         *ScriptAST
	Symbols     *SymbolTable
	Diagnostics Diagnostics
}

// ParseScript lexes and parses the source, returning the AST.
// Parsing continues even when errors are found (best-effort recovery).
func ParseScript(src string) *ParseResult {
	lex := NewLexer(src)
	tokens, lexErrors := lex.Tokenize()

	parser := NewParser(tokens)
	ast, parseErrors := parser.Parse()

	diags := append(Diagnostics{}, lexErrors...)
	diags = append(diags, parseErrors...)

	return &ParseResult{
		AST:         ast,
		Diagnostics: diags,
	}
}

// ExpandScript lexes, parses, and expands the source (resolves defaults,
// patterns, policies, links, scene ranges).
func ExpandScript(src string) *ParseResult {
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		return pr
	}

	expander := NewExpander(pr.AST)
	ast, expandErrors := expander.Expand()

	diags := append(pr.Diagnostics, expandErrors...)
	return &ParseResult{
		AST:         ast,
		Diagnostics: diags,
	}
}

// ValidateScript lexes, parses, expands and validates the source.
func ValidateScript(src string) *ValidateResult {
	pr := ExpandScript(src)
	if pr.Diagnostics.HasErrors() {
		return &ValidateResult{
			AST:         pr.AST,
			Diagnostics: pr.Diagnostics,
		}
	}

	validator := NewValidator(pr.AST)
	symbols, valErrors := validator.Validate()

	diags := append(pr.Diagnostics, valErrors...)
	return &ValidateResult{
		AST:         pr.AST,
		Symbols:     symbols,
		Diagnostics: diags,
	}
}

// CompileScript runs the full pipeline: lex → parse → expand → validate → compile.
// subscriberID and creatorID are required for entity generation.
func CompileScript(src string, subscriberID string, creatorID bson.ObjectId) *CompileResult {
	// Reset the counter so each compilation gets fresh unique public_ids.
	ResetIDCounter()

	vr := ValidateScript(src)
	if vr.Diagnostics.HasErrors() {
		return &CompileResult{
			Diagnostics: vr.Diagnostics,
		}
	}

	compiler := NewCompiler(vr.AST, vr.Symbols, subscriberID, creatorID)
	return compiler.Compile()
}
