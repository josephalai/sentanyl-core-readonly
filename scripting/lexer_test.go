package scripting

import (
	"testing"
)

// ---------- Basic Lexer Tests ----------

func TestLexerEmptyInput(t *testing.T) {
	lex := NewLexer("")
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if len(tokens) != 1 || tokens[0].Kind != TokEOF {
		t.Errorf("expected single EOF token, got %v", tokens)
	}
}

func TestLexerKeywords(t *testing.T) {
	src := `story storyline enactment scene on do when within`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	expected := []TokenKind{TokStory, TokStoryline, TokEnactment, TokScene, TokOn, TokDo, TokWhen, TokWithin, TokEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestLexerStringLiteral(t *testing.T) {
	src := `"Hello World" "escape\"test" "newline\n"`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if tokens[0].Literal != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", tokens[0].Literal)
	}
	if tokens[1].Literal != `escape"test` {
		t.Errorf("expected 'escape\"test', got %q", tokens[1].Literal)
	}
	if tokens[2].Literal != "newline\n" {
		t.Errorf("expected 'newline\\n', got %q", tokens[2].Literal)
	}
}

func TestLexerIntegers(t *testing.T) {
	src := `0 1 42 100`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	expected := []string{"0", "1", "42", "100"}
	for i, exp := range expected {
		if tokens[i].Kind != TokInt {
			t.Errorf("token %d: expected INT, got %s", i, tokens[i].Kind)
		}
		if tokens[i].Literal != exp {
			t.Errorf("token %d: expected %q, got %q", i, exp, tokens[i].Literal)
		}
	}
}

func TestLexerDurations(t *testing.T) {
	src := `1d 2h 30m 5s`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	expected := []struct {
		kind TokenKind
		lit  string
	}{
		{TokDuration, "1d"},
		{TokDuration, "2h"},
		{TokDuration, "30m"},
		{TokDuration, "5s"},
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp.kind {
			t.Errorf("token %d: expected %s, got %s (%q)", i, exp.kind, tokens[i].Kind, tokens[i].Literal)
		}
		if tokens[i].Literal != exp.lit {
			t.Errorf("token %d: expected %q, got %q", i, exp.lit, tokens[i].Literal)
		}
	}
}

func TestLexerBooleans(t *testing.T) {
	src := `true false`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if tokens[0].Kind != TokBool || tokens[0].Literal != "true" {
		t.Errorf("expected BOOL 'true', got %s %q", tokens[0].Kind, tokens[0].Literal)
	}
	if tokens[1].Kind != TokBool || tokens[1].Literal != "false" {
		t.Errorf("expected BOOL 'false', got %s %q", tokens[1].Kind, tokens[1].Literal)
	}
}

func TestLexerPunctuation(t *testing.T) {
	src := `{ } ( ) [ ] , . : = ! < > <= >= == != && ||`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	expected := []TokenKind{
		TokLBrace, TokRBrace, TokLParen, TokRParen, TokLBracket, TokRBracket,
		TokComma, TokDot, TokColon, TokEquals, TokBang, TokLT, TokGT,
		TokLTEQ, TokGTEQ, TokEQEQ, TokNEQ, TokAmpAmp, TokPipePipe, TokEOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestLexerComments(t *testing.T) {
	src := `story // this is a comment
storyline /* block comment */ enactment`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	expected := []TokenKind{TokStory, TokStoryline, TokEnactment, TokEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestLexerSourcePositions(t *testing.T) {
	src := "story\nstoryline"
	lex := NewLexer(src)
	tokens, _ := lex.Tokenize()
	if tokens[0].Pos.Line != 1 || tokens[0].Pos.Col != 1 {
		t.Errorf("expected story at 1:1, got %s", tokens[0].Pos)
	}
	if tokens[1].Pos.Line != 2 || tokens[1].Pos.Col != 1 {
		t.Errorf("expected storyline at 2:1, got %s", tokens[1].Pos)
	}
}

func TestLexerUnterminatedString(t *testing.T) {
	src := `"unterminated`
	lex := NewLexer(src)
	_, errs := lex.Tokenize()
	if len(errs) == 0 {
		t.Error("expected error for unterminated string")
	}
}

func TestLexerIllegalCharacter(t *testing.T) {
	src := `story @`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) == 0 {
		t.Error("expected error for illegal character")
	}
	// Should still tokenize the story keyword
	if tokens[0].Kind != TokStory {
		t.Errorf("expected first token to be 'story', got %s", tokens[0].Kind)
	}
}

func TestLexerAllEntityKeywords(t *testing.T) {
	src := `story storyline enactment scene message template email`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	expected := []TokenKind{TokStory, TokStoryline, TokEnactment, TokScene, TokMessage, TokTemplate, TokEmail, TokEOF}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestLexerAllTriggerKeywords(t *testing.T) {
	src := `click not_click open not_open sent webhook nothing bounce spam unsubscribe failure email_validated user_has_tag`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	expected := []TokenKind{
		TokClick, TokNotClick, TokOpen, TokNotOpen, TokSent, TokWebhook, TokNothing,
		TokBounce, TokSpam, TokUnsubscribe, TokFailure, TokEmailValidated, TokUserHasTag, TokEOF,
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s (%q)", i, exp, tokens[i].Kind, tokens[i].Literal)
		}
	}
}

func TestLexerAllActionKeywords(t *testing.T) {
	src := `next_scene prev_scene jump_to_enactment jump_to_storyline advance_to_next_storyline end_story mark_complete mark_failed retry_scene retry_enactment loop_to_enactment loop_to_storyline send_immediate wait`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	expected := []TokenKind{
		TokNextScene, TokPrevScene, TokJumpToEnactment, TokJumpToStoryline,
		TokAdvanceToNextStoryline, TokEndStory, TokMarkComplete, TokMarkFailed,
		TokRetryScene, TokRetryEnactment, TokLoopToEnactment, TokLoopToStoryline,
		TokSendImmediate, TokWait, TokEOF,
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s (%q)", i, exp, tokens[i].Kind, tokens[i].Literal)
		}
	}
}

func TestLexerConditionKeywords(t *testing.T) {
	src := `has_badge not_has_badge has_tag not_has_tag and or not`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	expected := []TokenKind{TokHasBadge, TokNotHasBadge, TokHasTag, TokNotHasTag, TokAnd, TokOr, TokNot, TokEOF}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestLexerIdentifier(t *testing.T) {
	src := `myVar some_thing _private AnotherName`
	lex := NewLexer(src)
	tokens, errs := lex.Tokenize()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	for i := 0; i < 4; i++ {
		if tokens[i].Kind != TokIdent {
			t.Errorf("token %d: expected IDENT, got %s (%q)", i, tokens[i].Kind, tokens[i].Literal)
		}
	}
}
