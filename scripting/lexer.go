package scripting

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenises a SentanylScript source string.
type Lexer struct {
	src    string
	pos    int  // current byte offset
	line   int  // 1-based line number
	col    int  // 1-based column (byte offset in current line)
	tokens []Token
	errors []Diagnostic
}

// NewLexer creates a Lexer for src.
func NewLexer(src string) *Lexer {
	return &Lexer{src: src, line: 1, col: 1}
}

// Tokenize scans the entire source and returns (tokens, errors).
// The token stream always ends with TokEOF.
func (l *Lexer) Tokenize() ([]Token, []Diagnostic) {
	for {
		tok := l.next()
		l.tokens = append(l.tokens, tok)
		if tok.Kind == TokEOF {
			break
		}
	}
	return l.tokens, l.errors
}

// ---------- internal helpers ----------

func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
	return r
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.src[l.pos:])
	l.pos += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col += size
	}
	return r
}

func (l *Lexer) curPos() Pos {
	return Pos{Line: l.line, Col: l.col, Offset: l.pos}
}

func (l *Lexer) makeToken(kind TokenKind, lit string, p Pos) Token {
	return Token{Kind: kind, Literal: lit, Pos: p}
}

func (l *Lexer) errorf(p Pos, format string, args ...interface{}) {
	l.errors = append(l.errors, Diagnostic{
		Pos:     p,
		Message: fmt.Sprintf(format, args...),
		Level:   DiagError,
	})
}

// skipWhitespaceAndComments consumes spaces, tabs, newlines, // comments, and # comments.
func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		r := l.peek()
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			l.advance()
			continue
		}
		// # line comment
		if r == '#' {
			for l.pos < len(l.src) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		// // line comment
		if r == '/' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '/' {
			l.advance() // first /
			l.advance() // second /
			for l.pos < len(l.src) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		// block comment
		if r == '/' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '*' {
			start := l.curPos()
			l.advance() // /
			l.advance() // *
			for l.pos < len(l.src) {
				if l.peek() == '*' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '/' {
					l.advance() // *
					l.advance() // /
					break
				}
				if l.pos >= len(l.src)-1 {
					l.errorf(start, "unterminated block comment")
					l.advance()
					break
				}
				l.advance()
			}
			continue
		}
		break
	}
}

// next returns the next token from the source.
func (l *Lexer) next() Token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.src) {
		return l.makeToken(TokEOF, "", l.curPos())
	}

	p := l.curPos()
	r := l.peek()

	// String literal
	if r == '"' {
		return l.readString(p)
	}

	// Number or duration
	if r >= '0' && r <= '9' {
		return l.readNumberOrDuration(p)
	}

	// Identifier / keyword (including underscore-joined keywords)
	if isIdentStart(r) {
		return l.readIdentOrKeyword(p)
	}

	// Two-character operators
	if l.pos+1 < len(l.src) {
		two := l.src[l.pos : l.pos+2]
		switch two {
		case "..":
			l.advance(); l.advance()
			return l.makeToken(TokDotDot, two, p)
		case "${":
			l.advance(); l.advance()
			return l.makeToken(TokDollarLBrace, two, p)
		case "<=":
			l.advance(); l.advance()
			return l.makeToken(TokLTEQ, two, p)
		case ">=":
			l.advance(); l.advance()
			return l.makeToken(TokGTEQ, two, p)
		case "==":
			l.advance(); l.advance()
			return l.makeToken(TokEQEQ, two, p)
		case "!=":
			l.advance(); l.advance()
			return l.makeToken(TokNEQ, two, p)
		case "&&":
			l.advance(); l.advance()
			return l.makeToken(TokAmpAmp, two, p)
		case "||":
			l.advance(); l.advance()
			return l.makeToken(TokPipePipe, two, p)
		}
	}

	// Single-character tokens
	l.advance()
	switch r {
	case '{':
		return l.makeToken(TokLBrace, "{", p)
	case '}':
		return l.makeToken(TokRBrace, "}", p)
	case '(':
		return l.makeToken(TokLParen, "(", p)
	case ')':
		return l.makeToken(TokRParen, ")", p)
	case '[':
		return l.makeToken(TokLBracket, "[", p)
	case ']':
		return l.makeToken(TokRBracket, "]", p)
	case ',':
		return l.makeToken(TokComma, ",", p)
	case '.':
		return l.makeToken(TokDot, ".", p)
	case ':':
		return l.makeToken(TokColon, ":", p)
	case '=':
		return l.makeToken(TokEquals, "=", p)
	case '!':
		return l.makeToken(TokBang, "!", p)
	case '<':
		return l.makeToken(TokLT, "<", p)
	case '>':
		return l.makeToken(TokGT, ">", p)
	case '+':
		return l.makeToken(TokPlus, "+", p)
	case '-':
		return l.makeToken(TokMinus, "-", p)
	case '*':
		return l.makeToken(TokStar, "*", p)
	case '/':
		return l.makeToken(TokSlash, "/", p)
	case '%':
		return l.makeToken(TokPercent, "%", p)
	}

	l.errorf(p, "unexpected character %q", r)
	return l.makeToken(TokIllegal, string(r), p)
}

// readString reads a double-quoted string literal. Supports basic escapes.
func (l *Lexer) readString(start Pos) Token {
	l.advance() // consume opening "
	var b strings.Builder
	for {
		if l.pos >= len(l.src) {
			l.errorf(start, "unterminated string literal")
			return l.makeToken(TokString, b.String(), start)
		}
		r := l.advance()
		if r == '"' {
			return l.makeToken(TokString, b.String(), start)
		}
		if r == '\\' {
			if l.pos >= len(l.src) {
				l.errorf(start, "unterminated escape in string")
				return l.makeToken(TokString, b.String(), start)
			}
			esc := l.advance()
			switch esc {
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte('\\')
				b.WriteRune(esc)
			}
			continue
		}
		b.WriteRune(r)
	}
}

// readNumberOrDuration reads an integer, float, or duration (e.g. 1d, 2h, 30m).
func (l *Lexer) readNumberOrDuration(start Pos) Token {
	numStart := l.pos
	for l.pos < len(l.src) && l.peek() >= '0' && l.peek() <= '9' {
		l.advance()
	}
	// Check for duration suffix
	if l.pos < len(l.src) {
		r := l.peek()
		if r == 'd' || r == 'h' || r == 'm' || r == 's' {
			// Could be a duration — peek further
			dStart := l.pos
			l.advance() // consume the unit character
			lit := l.src[numStart:l.pos]
			// Check for compound durations like 1h30m
			for l.pos < len(l.src) && l.peek() >= '0' && l.peek() <= '9' {
				l.advance()
				// after digits, expect another unit
				if l.pos < len(l.src) {
					r2 := l.peek()
					if r2 == 'd' || r2 == 'h' || r2 == 'm' || r2 == 's' {
						l.advance()
					}
				}
			}
			lit = l.src[numStart:l.pos]
			// Verify it's not actually an ident following a number
			if l.pos < len(l.src) && isIdentStart(l.peek()) {
				// backtrack — this is not a valid duration
				l.pos = dStart
				l.recalcLineCol(numStart)
				numStr := l.src[numStart:dStart]
				return l.makeToken(TokInt, numStr, start)
			}
			return l.makeToken(TokDuration, lit, start)
		}
	}
	// Check for float
	if l.pos < len(l.src) && l.peek() == '.' {
		// peek ahead for digit
		if l.pos+1 < len(l.src) && l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9' {
			l.advance() // consume .
			for l.pos < len(l.src) && l.peek() >= '0' && l.peek() <= '9' {
				l.advance()
			}
			return l.makeToken(TokFloat, l.src[numStart:l.pos], start)
		}
	}
	return l.makeToken(TokInt, l.src[numStart:l.pos], start)
}

// readIdentOrKeyword reads an identifier or keyword.
func (l *Lexer) readIdentOrKeyword(start Pos) Token {
	idStart := l.pos
	for l.pos < len(l.src) && isIdentContinue(l.peek()) {
		l.advance()
	}
	lit := l.src[idStart:l.pos]
	kind := LookupIdent(lit)
	// Map true/false to TokBool
	if kind == TokTrue || kind == TokFalse {
		return l.makeToken(TokBool, lit, start)
	}
	return l.makeToken(kind, lit, start)
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentContinue(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// recalcLineCol re-calculates line/col by scanning from the start to the given offset.
// Used only on backtrack which is rare.
func (l *Lexer) recalcLineCol(targetOffset int) {
	l.line = 1
	l.col = 1
	for i := 0; i < targetOffset && i < len(l.src); {
		r, size := utf8.DecodeRuneInString(l.src[i:])
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col += size
		}
		i += size
	}
}
