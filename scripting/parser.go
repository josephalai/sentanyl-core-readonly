package scripting

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser builds an AST from a token stream.
type Parser struct {
	tokens  []Token
	pos     int
	errors  Diagnostics
}

// NewParser creates a parser from the given token stream.
func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse parses the token stream and returns the AST root.
func (p *Parser) Parse() (*ScriptAST, Diagnostics) {
	ast := &ScriptAST{NodeBase: NodeBase{Pos: p.cur().Pos}}
	for !p.atEnd() {
		switch {
		case p.check(TokDefault):
			ds := p.parseDefaultSender()
			if ds != nil {
				ast.DefaultSenders = append(ast.DefaultSenders, ds)
			}
		case p.check(TokLinks):
			ast.Links = p.parseLinksBlock()
		case p.check(TokPattern):
			pat := p.parsePatternDef()
			if pat != nil {
				ast.Patterns = append(ast.Patterns, pat)
			}
		case p.check(TokPolicy):
			pol := p.parsePolicyDef()
			if pol != nil {
				ast.Policies = append(ast.Policies, pol)
			}
		case p.check(TokData):
			db := p.parseDataBlock()
			if db != nil {
				ast.DataBlocks = append(ast.DataBlocks, db)
			}
		case p.check(TokSceneDefaults):
			ast.SceneDefaults = p.parseSceneDefaults()
		case p.check(TokEnactmentDefaults):
			ast.EnactmentDefaults = p.parseEnactmentDefaults()
		case p.check(TokStory):
			story := p.parseStory()
			if story != nil {
				ast.Stories = append(ast.Stories, story)
			}
		case p.check(TokFunnel):
			f := p.parseFunnel()
			if f != nil {
				ast.Funnels = append(ast.Funnels, f)
			}
		case p.check(TokCourse):
			cd := p.parseCourseDecl()
			if cd != nil {
				ast.Courses = append(ast.Courses, cd)
			}
		case p.check(TokProduct):
			pd := p.parseProductDecl()
			if pd != nil {
				ast.Products = append(ast.Products, pd)
			}
		case p.check(TokOffer):
			od := p.parseOfferDecl()
			if od != nil {
				ast.Offers = append(ast.Offers, od)
			}
		case p.check(TokQuiz):
			qz := p.parseQuiz()
			if qz != nil {
				ast.Quizzes = append(ast.Quizzes, qz)
			}
		case p.check(TokSite):
			s := p.parseSite()
			if s != nil {
				ast.Sites = append(ast.Sites, s)
			}
		case p.check(TokMedia):
			m := p.parseMediaDecl()
			if m != nil {
				ast.MediaDecls = append(ast.MediaDecls, m)
			}
		case p.check(TokPlayerPreset):
			pp := p.parsePlayerPresetDecl()
			if pp != nil {
				ast.PlayerPresets = append(ast.PlayerPresets, pp)
			}
		case p.check(TokChannel):
			ch := p.parseChannelDecl()
			if ch != nil {
				ast.ChannelDecls = append(ast.ChannelDecls, ch)
			}
		case p.check(TokMediaWebhook):
			wh := p.parseMediaWebhookDecl()
			if wh != nil {
				ast.MediaWebhookDecls = append(ast.MediaWebhookDecls, wh)
			}
		case p.check(TokCampaign):
			cm := p.parseCampaign()
			if cm != nil {
				ast.Campaigns = append(ast.Campaigns, cm)
			}
		default:
			p.errorf("expected 'story', 'funnel', 'site', 'course', 'product', 'offer', 'quiz', 'media', 'player_preset', 'channel', 'media_webhook', 'campaign', 'default', 'links', 'pattern', 'policy', 'data', 'scene_defaults', or 'enactment_defaults', got %s", p.cur().Kind)
			p.advance()
		}
	}
	return ast, p.errors
}

// ---------- Helpers ----------

func (p *Parser) cur() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: TokEOF, Pos: Pos{}}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Kind: TokEOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) atEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Kind == TokEOF
}

func (p *Parser) advance() Token {
	tok := p.cur()
	if !p.atEnd() {
		p.pos++
	}
	return tok
}

func (p *Parser) check(kinds ...TokenKind) bool {
	cur := p.cur().Kind
	for _, k := range kinds {
		if cur == k {
			return true
		}
	}
	return false
}

func (p *Parser) expect(kind TokenKind) Token {
	if p.cur().Kind != kind {
		p.errorf("expected %s, got %s (%q)", kind, p.cur().Kind, p.cur().Literal)
		return Token{Kind: TokIllegal, Pos: p.cur().Pos}
	}
	return p.advance()
}

func (p *Parser) expectString() string {
	tok := p.expect(TokString)
	return tok.Literal
}

func (p *Parser) expectInt() int {
	tok := p.expect(TokInt)
	v, err := strconv.Atoi(tok.Literal)
	if err != nil {
		p.errorf("invalid integer %q", tok.Literal)
	}
	return v
}

func (p *Parser) expectFloat() float64 {
	tok := p.expect(TokFloat)
	v, err := strconv.ParseFloat(tok.Literal, 64)
	if err != nil {
		p.errorf("invalid float %q", tok.Literal)
	}
	return v
}

func (p *Parser) expectBool() bool {
	tok := p.cur()
	if tok.Kind == TokBool || tok.Kind == TokTrue || tok.Kind == TokFalse {
		p.advance()
		return tok.Literal == "true"
	}
	p.errorf("expected boolean, got %s", tok.Kind)
	p.advance()
	return false
}

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, Diagnostic{
		Pos:     p.cur().Pos,
		Message: fmt.Sprintf(format, args...),
		Level:   DiagError,
	})
}

// isIdentLike returns true if the current token can be used as a parameter/variable name.
// This includes TokIdent plus any keyword that is a valid identifier (like "name", "link", etc.)
func (p *Parser) isIdentLike() bool {
	k := p.cur().Kind
	if k == TokIdent {
		return true
	}
	// Allow keywords as parameter names in pattern/policy definitions
	switch k {
	case TokName, TokLinks, TokBody, TokSubject, TokOrder, TokLevel,
		TokPriority, TokMessage, TokTemplate, TokEmail, TokRoute, TokBadge,
		TokScene, TokTags, TokVars, TokData, TokDomain, TokField, TokType,
		TokPage, TokBlock, TokForm, TokStage, TokPath, TokLength, TokPrompt,
		TokRequired, TokNew, TokPdf, TokLeadMagnet, TokReference, TokContext, TokGlobal, TokExtend, TokFunnel,
		TokProduct, TokOffer, TokPrice, TokCurrency, TokQuestion, TokAnswer,
		TokQuiz, TokScore, TokCustom, TokSite, TokSEO, TokNavigation, TokHeader,
		TokFooter, TokTheme, TokTitle, TokDescription, TokModule, TokLesson,
		TokVideoURL, TokContent, TokDraft,
		TokInstructor, TokLMSDuration, TokIsFree, TokIsDraft, TokDripDays, TokDripHours, TokDripMinutes,
		TokContentGen, TokDescriptionGen, TokPassThreshold, TokMaxAttempts,
		TokOptions, TokMultipleChoice, TokShortAnswer, TokCertificate, TokCourseRef,
		TokCourse, TokDurationKw, TokAudience, TokOutcome, TokTone, TokDefaultMedia,
		TokMedia, TokPlayerPreset, TokChannel, TokChapter, TokTurnstile,
		TokCTA, TokAnnotation, TokBadgeRule, TokMediaWebhook, TokMediaRef,
		TokStartSec, TokEndSec, TokThreshold, TokOperator, TokEnabled,
		TokProgress, TokComplete, TokPlay, TokPause, TokRewatch,
		TokPosterURL, TokPlayerColor:
		return true
	}
	return false
}

// advanceIdentLike consumes the current token as an identifier-like name.
func (p *Parser) advanceIdentLike() string {
	tok := p.advance()
	return tok.Literal
}

// expectStringOrIdent consumes a string literal or identifier-like token.
// Used in contexts where pattern parameter references may appear as bare identifiers.
func (p *Parser) expectStringOrIdent() string {
	if p.check(TokString) {
		return p.expectString()
	}
	if p.isIdentLike() {
		return p.advanceIdentLike()
	}
	p.errorf("expected string or identifier, got %s", p.cur().Kind)
	p.advance()
	return ""
}

// parseTriggerActionRef parses an identifier that may include dot-access (e.g., ws.info).
// Returns "ident" or "ident.field" as a string for later substitution by the expander.
func (p *Parser) parseTriggerActionRef() string {
	name := p.advance().Literal // consume the first identifier
	// Check for dot-access: ident.field
	if p.check(TokDot) {
		p.advance() // consume '.'
		if p.check(TokIdent) || p.isIdentLike() {
			name += "." + p.advance().Literal
		} else {
			p.errorf("expected field name after '.', got %s", p.cur().Kind)
		}
	}
	return name
}

// ---------- DSL v2 Top-Level Parsing ----------

// parseDefaultSender parses: default sender { from_email "..." from_name "..." reply_to "..." }
// Also supports named form: default sender "name" { ... }
func (p *Parser) parseDefaultSender() *DefaultSenderNode {
	pos := p.cur().Pos
	p.expect(TokDefault)
	p.expect(TokSender)

	node := &DefaultSenderNode{NodeBase: NodeBase{Pos: pos}}

	// Optional name
	if p.check(TokString) {
		node.Name = p.expectString()
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokFromEmail:
			p.advance()
			node.FromEmail = p.expectString()
		case TokFromName:
			p.advance()
			node.FromName = p.expectString()
		case TokReplyTo:
			p.advance()
			node.ReplyTo = p.expectString()
		default:
			p.errorf("unexpected token %s in default sender block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseLinksBlock parses: links { name = "url" ... }
func (p *Parser) parseLinksBlock() *LinksBlockNode {
	pos := p.cur().Pos
	p.expect(TokLinks)

	node := &LinksBlockNode{
		NodeBase: NodeBase{Pos: pos},
		Links:    make(map[string]string),
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		// Expect: identifier = "string"
		if p.check(TokIdent) {
			name := p.advance().Literal
			p.expect(TokEquals)
			value := p.expectString()
			node.Links[name] = value
		} else {
			p.errorf("expected link name identifier in links block, got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parsePatternDef parses: pattern name(param1, param2, ...) { enactment/scenes ... }
func (p *Parser) parsePatternDef() *PatternDefNode {
	pos := p.cur().Pos
	p.expect(TokPattern)

	// Pattern name (identifier)
	nameTok := p.cur()
	if !p.check(TokIdent) {
		p.errorf("expected pattern name, got %s", p.cur().Kind)
		return nil
	}
	p.advance()

	node := &PatternDefNode{
		NodeBase: NodeBase{Pos: pos},
		Name:     nameTok.Literal,
	}

	// Parameters: (param1, param2, ...)
	p.expect(TokLParen)
	for !p.check(TokRParen) && !p.atEnd() {
		if p.isIdentLike() {
			node.Params = append(node.Params, p.advanceIdentLike())
		} else if p.check(TokComma) {
			p.advance()
		} else {
			p.errorf("expected parameter name or ')', got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRParen)
	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokEnactment:
			en := p.parseEnactment()
			if en != nil {
				node.Enactments = append(node.Enactments, en)
			}
		case TokScene:
			sc := p.parseScene()
			if sc != nil {
				node.Scenes = append(node.Scenes, sc)
			}
		case TokScenes:
			node.ScenesRange = p.parseScenesRange()
		case TokOn:
			tr := p.parseTrigger()
			if tr != nil {
				node.Triggers = append(node.Triggers, tr)
			}
		case TokUse:
			// Allow nested use statements within patterns
			p.errorf("nested 'use' in pattern definitions is not supported")
			p.advance()
		default:
			p.errorf("unexpected token %s in pattern body", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parsePolicyDef parses: policy name(param1, param2, ...) { on ... }
func (p *Parser) parsePolicyDef() *PolicyDefNode {
	pos := p.cur().Pos
	p.expect(TokPolicy)

	// Policy name (identifier)
	nameTok := p.cur()
	if !p.check(TokIdent) {
		p.errorf("expected policy name, got %s", p.cur().Kind)
		return nil
	}
	p.advance()

	node := &PolicyDefNode{
		NodeBase: NodeBase{Pos: pos},
		Name:     nameTok.Literal,
	}

	// Parameters: (param1, param2, ...)
	p.expect(TokLParen)
	for !p.check(TokRParen) && !p.atEnd() {
		if p.isIdentLike() {
			node.Params = append(node.Params, p.advanceIdentLike())
		} else if p.check(TokComma) {
			p.advance()
		} else {
			p.errorf("expected parameter name or ')', got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRParen)

	// Body: { trigger definitions }
	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOn:
			tr := p.parseTrigger()
			if tr != nil {
				node.Triggers = append(node.Triggers, tr)
			}
		default:
			p.errorf("unexpected token %s in policy body (expected 'on')", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseScenesRange parses: scenes 1..3 as var_name { scene "..." { ... } }
func (p *Parser) parseScenesRange() *ScenesRangeNode {
	pos := p.cur().Pos
	p.expect(TokScenes) // consume 'scenes'

	node := &ScenesRangeNode{NodeBase: NodeBase{Pos: pos}}

	// Range: start..end
	node.RangeStart = p.expectInt()
	p.expect(TokDotDot)
	node.RangeEnd = p.expectInt()

	// as variable_name
	p.expect(TokAs)
	if p.check(TokIdent) {
		node.Variable = p.advance().Literal
	} else {
		p.errorf("expected variable name after 'as', got %s", p.cur().Kind)
	}

	// Body: { scene template }
	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		if p.check(TokScene) {
			sc := p.parseScene()
			if sc != nil {
				node.Body = sc
			}
		} else {
			p.errorf("expected 'scene' in scenes range body, got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- DSL v3: Data Blocks ----------

// parseDataBlock parses: data name = [{ key: val }, ...]
func (p *Parser) parseDataBlock() *DataBlockNode {
	pos := p.cur().Pos
	p.expect(TokData)

	node := &DataBlockNode{NodeBase: NodeBase{Pos: pos}}

	// Data block name
	if p.isIdentLike() {
		node.Name = p.advanceIdentLike()
	} else {
		p.errorf("expected data block name, got %s", p.cur().Kind)
		return nil
	}

	p.expect(TokEquals)
	p.expect(TokLBracket)

	// Parse array of object literals: [{ ... }, { ... }, ...]
	for !p.check(TokRBracket) && !p.atEnd() {
		if p.check(TokLBrace) {
			obj := p.parseDataObjectLiteral()
			if obj != nil {
				node.Items = append(node.Items, obj)
			}
		} else if p.check(TokComma) {
			p.advance()
		} else {
			p.errorf("expected '{' or ']' in data array, got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBracket)
	return node
}

// parseDataObjectLiteral parses: { key: val, key2: val2 }
// Values can be strings ("...") or identifiers (link references, etc.)
func (p *Parser) parseDataObjectLiteral() *DataObjectLiteral {
	pos := p.cur().Pos
	p.expect(TokLBrace)

	obj := &DataObjectLiteral{
		NodeBase: NodeBase{Pos: pos},
		Fields:   make(map[string]string),
	}

	for !p.check(TokRBrace) && !p.atEnd() {
		// key: value
		var key string
		if p.isIdentLike() {
			key = p.advanceIdentLike()
		} else {
			p.errorf("expected field name in data object, got %s", p.cur().Kind)
			p.advance()
			continue
		}

		p.expect(TokColon)

		// Value: string, identifier, or integer
		var val string
		if p.check(TokString) {
			val = p.expectString()
		} else if p.check(TokInt) {
			val = p.advance().Literal
		} else if p.isIdentLike() {
			val = p.advanceIdentLike()
		} else {
			p.errorf("expected string, integer, or identifier value in data object, got %s", p.cur().Kind)
			p.advance()
			continue
		}

		obj.Fields[key] = val

		// Optional comma
		if p.check(TokComma) {
			p.advance()
		}
	}

	p.expect(TokRBrace)
	return obj
}

// ---------- DSL v3: For Loop Parsing ----------

// parseForLoop parses: for var in data_ref { ... } or for var in [{ ... }, ...] { ... }
// context is "story" (generates storylines) or "storyline" (generates enactments)
func (p *Parser) parseForLoop(context string) *ForNode {
	pos := p.cur().Pos
	p.expect(TokFor)

	node := &ForNode{NodeBase: NodeBase{Pos: pos}}

	// Loop variable name
	if p.isIdentLike() {
		node.Variable = p.advanceIdentLike()
	} else {
		p.errorf("expected loop variable name after 'for', got %s", p.cur().Kind)
		return nil
	}

	p.expect(TokIn)

	// Data source: identifier reference to data block OR inline array
	if p.check(TokLBracket) {
		// Inline array of objects: [{ ... }, { ... }]
		p.advance()
		for !p.check(TokRBracket) && !p.atEnd() {
			if p.check(TokLBrace) {
				obj := p.parseDataObjectLiteral()
				if obj != nil {
					node.ObjectItems = append(node.ObjectItems, obj)
				}
			} else if p.check(TokString) {
				// Simple string list
				node.Items = append(node.Items, p.expectString())
			} else if p.check(TokComma) {
				p.advance()
			} else {
				p.errorf("expected '{', string, or ']' in for loop data, got %s", p.cur().Kind)
				p.advance()
			}
		}
		p.expect(TokRBracket)
	} else if p.isIdentLike() {
		// Reference to a named data block
		node.DataRef = p.advanceIdentLike()
	} else {
		p.errorf("expected data reference or inline array after 'in', got %s", p.cur().Kind)
		return nil
	}

	// Body: { ... }
	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch context {
		case "story":
			// Inside story: for loop generates storylines
			switch p.cur().Kind {
			case TokStoryline:
				sl := p.parseStoryline()
				if sl != nil {
					node.Body = append(node.Body, sl)
				}
			default:
				p.errorf("expected 'storyline' in for loop body (story context), got %s", p.cur().Kind)
				p.advance()
			}
		case "storyline":
			// Inside storyline: for loop generates enactments or use pattern
			switch p.cur().Kind {
			case TokEnactment:
				en := p.parseEnactment()
				if en != nil {
					node.BodyEnactments = append(node.BodyEnactments, en)
				}
			case TokUse:
				us := p.parseUseStatement()
				if us != nil {
					node.BodyUseStatements = append(node.BodyUseStatements, us)
				}
			default:
				p.errorf("expected 'enactment' or 'use' in for loop body (storyline context), got %s", p.cur().Kind)
				p.advance()
			}
		default:
			p.errorf("unsupported for loop context: %s", context)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- DSL v3: Defaults Blocks ----------

// parseSceneDefaults parses: scene_defaults { on ... use policy ... }
func (p *Parser) parseSceneDefaults() *SceneDefaultsNode {
	pos := p.cur().Pos
	p.expect(TokSceneDefaults)

	node := &SceneDefaultsNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOn:
			tr := p.parseTrigger()
			if tr != nil {
				node.Triggers = append(node.Triggers, tr)
			}
		case TokUse:
			us := p.parseUseStatement()
			if us != nil {
				node.UseStatements = append(node.UseStatements, us)
			}
		default:
			p.errorf("expected 'on' or 'use' in scene_defaults block, got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseEnactmentDefaults parses: enactment_defaults { on ... use policy ... }
func (p *Parser) parseEnactmentDefaults() *EnactmentDefaultsNode {
	pos := p.cur().Pos
	p.expect(TokEnactmentDefaults)

	node := &EnactmentDefaultsNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOn:
			tr := p.parseTrigger()
			if tr != nil {
				node.Triggers = append(node.Triggers, tr)
			}
		case TokUse:
			us := p.parseUseStatement()
			if us != nil {
				node.UseStatements = append(node.UseStatements, us)
			}
		default:
			p.errorf("expected 'on' or 'use' in enactment_defaults block, got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseUseStatement parses: use pattern/policy/sender name(args...)
// or: use sender default
func (p *Parser) parseUseStatement() *UseStatementNode {
	pos := p.cur().Pos
	p.expect(TokUse)

	node := &UseStatementNode{NodeBase: NodeBase{Pos: pos}}

	// Kind: pattern, policy, or sender
	switch p.cur().Kind {
	case TokPattern:
		p.advance()
		node.Kind = "pattern"
	case TokPolicy:
		p.advance()
		node.Kind = "policy"
	case TokSender:
		p.advance()
		node.Kind = "sender"
		// "use sender default" or "use sender name"
		if p.check(TokDefault) {
			node.Target = "default"
			p.advance()
		} else if p.check(TokIdent) {
			node.Target = p.advance().Literal
		} else if p.check(TokString) {
			node.Target = p.expectString()
		}
		return node
	default:
		p.errorf("expected 'pattern', 'policy', or 'sender' after 'use', got %s", p.cur().Kind)
		p.advance()
		return nil
	}

	// Target name
	if p.isIdentLike() {
		node.Target = p.advanceIdentLike()
	} else {
		p.errorf("expected name after 'use %s', got %s", node.Kind, p.cur().Kind)
		return nil
	}

	// Arguments: (arg1, arg2, ...)
	if p.check(TokLParen) {
		p.advance()
		for !p.check(TokRParen) && !p.atEnd() {
			if p.check(TokString) {
				node.Args = append(node.Args, p.expectString())
			} else if p.isIdentLike() {
				// Allow identifier references (e.g. link names, dot-access like phase.link)
				arg := p.advanceIdentLike()
				// Check for dot-access: var.field
				for p.check(TokDot) {
					p.advance()
					if p.isIdentLike() {
						arg += "." + p.advanceIdentLike()
					} else {
						p.errorf("expected field name after '.' in argument, got %s", p.cur().Kind)
						break
					}
				}
				node.Args = append(node.Args, arg)
			} else if p.check(TokComma) {
				p.advance()
			} else {
				p.errorf("expected argument in use statement, got %s", p.cur().Kind)
				p.advance()
			}
		}
		p.expect(TokRParen)
	}

	return node
}

// ---------- Story Parsing ----------

func (p *Parser) parseStory() *StoryNode {
	pos := p.cur().Pos
	p.expect(TokStory)
	name := p.expectString()
	node := &StoryNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokPriority:
			p.advance()
			v := p.expectInt()
			node.Priority = &v
		case TokAllowInterruption:
			p.advance()
			v := p.expectBool()
			node.AllowInterruption = &v
		case TokOnBegin:
			node.OnBegin = p.parseLifecycleBlock()
		case TokOnComplete:
			node.OnComplete = p.parseOnCompleteBlock()
		case TokOnFail:
			node.OnFail = p.parseOnFailBlock()
		case TokRequiredBadges:
			node.RequiredBadges = p.parseRequiredBadges()
		case TokStartTrigger:
			p.advance()
			v := p.expectString()
			node.StartTrigger = &v
		case TokCompleteTrigger:
			p.advance()
			v := p.expectString()
			node.CompleteTrigger = &v
		case TokStoryline:
			sl := p.parseStoryline()
			if sl != nil {
				node.Storylines = append(node.Storylines, sl)
			}
		case TokFor:
			fl := p.parseForLoop("story")
			if fl != nil {
				node.ForLoops = append(node.ForLoops, fl)
			}
		case TokUse:
			us := p.parseUseStatement()
			if us != nil {
				node.UseStatements = append(node.UseStatements, us)
			}
		default:
			p.errorf("unexpected token %s in story block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Storyline Parsing ----------

func (p *Parser) parseStoryline() *StorylineNode {
	pos := p.cur().Pos
	p.expect(TokStoryline)
	name := p.expectString()
	node := &StorylineNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOrder:
			p.advance()
			if p.check(TokInt) {
				v := p.expectInt()
				node.Order = &v
			} else if p.check(TokIdent) || p.isIdentLike() {
				node.OrderExpr = p.parseTriggerActionRef()
			} else {
				p.errorf("expected integer or expression after 'order', got %s", p.cur().Kind)
				p.advance()
			}
		case TokRequiredBadges:
			node.RequiredBadges = p.parseRequiredBadges()
		case TokOnBegin:
			node.OnBegin = p.parseLifecycleBlock()
		case TokOnComplete:
			node.OnComplete = p.parseStorylineCompleteBlock()
		case TokOnFail:
			node.OnFail = p.parseStorylineFailBlock()
		case TokEnactment:
			en := p.parseEnactment()
			if en != nil {
				node.Enactments = append(node.Enactments, en)
			}
		case TokFor:
			fl := p.parseForLoop("storyline")
			if fl != nil {
				node.ForLoops = append(node.ForLoops, fl)
			}
		case TokUse:
			us := p.parseUseStatement()
			if us != nil {
				node.UseStatements = append(node.UseStatements, us)
			}
		default:
			p.errorf("unexpected token %s in storyline block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Enactment Parsing ----------

func (p *Parser) parseEnactment() *EnactmentNode {
	pos := p.cur().Pos
	p.expect(TokEnactment)
	// Accept string literal or bare identifier (for pattern parameter references)
	var name string
	if p.check(TokString) {
		name = p.expectString()
	} else if p.isIdentLike() {
		name = p.advanceIdentLike()
	} else {
		p.errorf("expected enactment name (string or identifier), got %s", p.cur().Kind)
		name = ""
		p.advance()
	}
	node := &EnactmentNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokLevel:
			p.advance()
			if p.check(TokInt) {
				v := p.expectInt()
				node.Level = &v
			} else if p.check(TokIdent) || p.isIdentLike() {
				node.LevelExpr = p.parseTriggerActionRef()
			} else {
				p.errorf("expected integer or expression after 'level', got %s", p.cur().Kind)
				p.advance()
			}
		case TokOrder:
			p.advance()
			if p.check(TokInt) {
				v := p.expectInt()
				node.Order = &v
			} else if p.check(TokIdent) || p.isIdentLike() {
				node.OrderExpr = p.parseTriggerActionRef()
			} else {
				p.errorf("expected integer or expression after 'order', got %s", p.cur().Kind)
				p.advance()
			}
		case TokSkipToNextStorylineOnExpiry:
			p.advance()
			v := p.expectBool()
			node.SkipToNextStorylineOnExpiry = &v
		case TokScene:
			sc := p.parseScene()
			if sc != nil {
				node.Scenes = append(node.Scenes, sc)
			}
		case TokOn:
			tr := p.parseTrigger()
			if tr != nil {
				node.Triggers = append(node.Triggers, tr)
			}
		case TokUse:
			us := p.parseUseStatement()
			if us != nil {
				node.UseStatements = append(node.UseStatements, us)
			}
		case TokScenes:
			node.ScenesRange = p.parseScenesRange()
		default:
			p.errorf("unexpected token %s in enactment block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Scene Parsing ----------

func (p *Parser) parseScene() *SceneNode {
	pos := p.cur().Pos
	p.expect(TokScene)
	name := p.expectString()
	node := &SceneNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokSubject:
			p.advance()
			node.Subject = p.expectStringOrIdent()
		case TokBody:
			p.advance()
			node.Body = p.expectStringOrIdent()
		case TokFromEmail:
			p.advance()
			node.FromEmail = p.expectStringOrIdent()
		case TokFromName:
			p.advance()
			node.FromName = p.expectStringOrIdent()
		case TokReplyTo:
			p.advance()
			node.ReplyTo = p.expectStringOrIdent()
		case TokTemplate:
			p.advance()
			node.TemplateName = p.expectString()
		case TokVars:
			p.advance()
			node.Vars = p.parseVarsMap()
		case TokTags:
			p.advance()
			node.Tags = p.parseStringList()
		case TokCertificate:
			node.Certificate = p.parseCertificate()
		case TokContextPack:
			p.advance()
			packID := p.expectString()
			node.ContextPackRefs = append(node.ContextPackRefs, packID)
		case TokSubjectGen:
			p.advance()
			node.SubjectGen = p.expectString()
		case TokBodyGen:
			p.advance()
			node.BodyGen = p.expectString()
		default:
			p.errorf("unexpected token %s in scene block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Campaign Parsing ----------

// parseCampaign parses: campaign "Name" { subject "..." body "..." subject_gen "..."
//   body_gen "..." context_pack "id" from_email "..." from_name "..." reply_to "..."
//   audience { must_have ["b1"] must_not_have ["b2"] }
//   on_click "<url-or-pattern>" { award_badge "<name>" }
// }
func (p *Parser) parseCampaign() *CampaignNode {
	pos := p.cur().Pos
	p.expect(TokCampaign)
	name := p.expectString()
	node := &CampaignNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokSubject:
			p.advance()
			node.Subject = p.expectStringOrIdent()
		case TokBody:
			p.advance()
			node.Body = p.expectStringOrIdent()
		case TokFromEmail:
			p.advance()
			node.FromEmail = p.expectStringOrIdent()
		case TokFromName:
			p.advance()
			node.FromName = p.expectStringOrIdent()
		case TokReplyTo:
			p.advance()
			node.ReplyTo = p.expectStringOrIdent()
		case TokContextPack:
			p.advance()
			node.ContextPackRefs = append(node.ContextPackRefs, p.expectString())
		case TokSubjectGen:
			p.advance()
			node.SubjectGen = p.expectString()
		case TokBodyGen:
			p.advance()
			node.BodyGen = p.expectString()
		case TokAudience:
			node.Audience = p.parseCampaignAudience()
		case TokOnClick:
			rule := p.parseCampaignOnClick()
			if rule != nil {
				node.OnClick = append(node.OnClick, rule)
			}
		default:
			p.errorf("unexpected token %s in campaign block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseCampaignAudience parses: audience { must_have ["b1","b2"] must_not_have ["b3"] }
func (p *Parser) parseCampaignAudience() *CampaignAudienceNode {
	pos := p.cur().Pos
	p.advance() // consume audience
	node := &CampaignAudienceNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokMustHave:
			p.advance()
			node.MustHave = append(node.MustHave, p.parseStringListOrSingle()...)
		case TokMustNotHave:
			p.advance()
			node.MustNotHave = append(node.MustNotHave, p.parseStringListOrSingle()...)
		default:
			p.errorf("unexpected token %s in audience block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseCampaignOnClick parses: on_click "<url-or-pattern>" { give_badge "<name>" }
// give_badge is the existing token; we treat it as the v1 award action.
func (p *Parser) parseCampaignOnClick() *CampaignClickRuleNode {
	pos := p.cur().Pos
	p.expect(TokOnClick)
	urlPat := ""
	if p.check(TokString) {
		urlPat = p.expectString()
	}
	rule := &CampaignClickRuleNode{NodeBase: NodeBase{Pos: pos}, URLPattern: urlPat}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokGiveBadge:
			p.advance()
			rule.AwardBadge = p.expectString()
		default:
			p.errorf("unexpected token %s in on_click block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return rule
}

// ---------- Trigger Parsing ----------

func (p *Parser) parseTrigger() *TriggerNode {
	pos := p.cur().Pos
	p.expect(TokOn)

	node := &TriggerNode{NodeBase: NodeBase{Pos: pos}}

	// Parse trigger type
	switch p.cur().Kind {
	case TokClick:
		p.advance()
		node.TriggerType = "click"
		// Optional action value (link URL, parameter name, or dot-access like ws.info)
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		} else if p.check(TokIdent) || p.isIdentLike() {
			node.UserActionValue = p.parseTriggerActionRef()
		}
	case TokNotClick:
		p.advance()
		node.TriggerType = "not_click"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		} else if p.check(TokIdent) || p.isIdentLike() {
			node.UserActionValue = p.parseTriggerActionRef()
		}
	case TokOpen:
		p.advance()
		node.TriggerType = "open"
	case TokNotOpen:
		p.advance()
		node.TriggerType = "not_open"
	case TokSent:
		p.advance()
		node.TriggerType = "sent"
	case TokWebhook:
		p.advance()
		node.TriggerType = "webhook"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		} else if p.check(TokIdent) {
			node.UserActionValue = p.advance().Literal
		}
	case TokNothing:
		p.advance()
		node.TriggerType = "nothing"
	case TokElse:
		p.advance()
		node.TriggerType = "else"
	case TokBounce:
		p.advance()
		node.TriggerType = "bounce"
	case TokSpam:
		p.advance()
		node.TriggerType = "spam"
	case TokUnsubscribe:
		p.advance()
		node.TriggerType = "unsubscribe"
	case TokFailure:
		p.advance()
		node.TriggerType = "failure"
	case TokEmailValidated:
		p.advance()
		node.TriggerType = "email_validated"
	case TokUserHasTag:
		p.advance()
		node.TriggerType = "user_has_tag"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokBadge:
		p.advance()
		node.TriggerType = "badge"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokSubmit:
		p.advance()
		node.TriggerType = "submit"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokAbandon:
		p.advance()
		node.TriggerType = "abandon"
	case TokPurchase:
		p.advance()
		node.TriggerType = "purchase"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokDecline:
		p.advance()
		node.TriggerType = "decline"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokWatch:
		p.advance()
		node.TriggerType = "watch"
		if p.check(TokString) {
			node.WatchBlockID = p.expectString()
		}
		if p.check(TokGT) || p.check(TokLT) || p.check(TokGTEQ) || p.check(TokLTEQ) {
			node.WatchOperator = p.advance().Literal
		}
		if p.check(TokInt) {
			node.WatchPercent = p.expectInt()
		}
		if p.check(TokPercent) {
			p.advance()
		}
	case TokPlay:
		p.advance()
		node.TriggerType = "play"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokPause:
		p.advance()
		node.TriggerType = "pause"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokProgress:
		p.advance()
		node.TriggerType = "progress"
		if p.check(TokString) {
			node.WatchBlockID = p.expectString()
		}
		if p.check(TokGT) || p.check(TokLT) || p.check(TokGTEQ) || p.check(TokLTEQ) {
			node.WatchOperator = p.advance().Literal
		}
		if p.check(TokInt) {
			node.WatchPercent = p.expectInt()
		}
		if p.check(TokPercent) {
			p.advance()
		}
	case TokComplete:
		p.advance()
		node.TriggerType = "complete"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokRewatch:
		p.advance()
		node.TriggerType = "rewatch"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokCTA:
		p.advance()
		node.TriggerType = "cta_click"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokTurnstile:
		p.advance()
		node.TriggerType = "turnstile_submit"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	case TokChapter:
		p.advance()
		node.TriggerType = "chapter_click"
		if p.check(TokString) {
			node.UserActionValue = p.expectString()
		}
	default:
		p.errorf("expected trigger type after 'on', got %s", p.cur().Kind)
		p.advance()
		return nil
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokWithin:
			p.advance()
			node.Within = p.parseDuration()
		case TokTriggerPriority:
			p.advance()
			v := p.expectInt()
			node.Priority = &v
		case TokPersistScope:
			p.advance()
			node.PersistScope = p.expectString()
		case TokMarkComplete:
			p.advance()
			node.MarkComplete = p.expectBool()
		case TokMarkFailed:
			p.advance()
			node.MarkFailed = p.expectBool()
		case TokRequiredBadges:
			node.RequiredBadges = p.parseRequiredBadges()
		case TokWhen:
			p.advance()
			cond := p.parseCondition()
			if cond != nil {
				node.Conditions = append(node.Conditions, cond)
			}
		case TokDo:
			p.advance()
			action := p.parseAction()
			if action != nil {
				node.Actions = append(node.Actions, action)
			}
		case TokElse:
			p.advance()
			// else { ... } block
			if p.check(TokLBrace) {
				p.expect(TokLBrace)
				for !p.check(TokRBrace) && !p.atEnd() {
					if p.check(TokDo) {
						p.advance()
					}
					action := p.parseAction()
					if action != nil {
						node.ElseActions = append(node.ElseActions, action)
					}
				}
				p.expect(TokRBrace)
			} else if p.check(TokDo) {
				// else do <action>
				p.advance()
				action := p.parseAction()
				if action != nil {
					node.ElseActions = append(node.ElseActions, action)
				}
			}
		case TokSendImmediate:
			p.advance()
			v := p.expectBool()
			// Apply to all actions or create a placeholder
			for _, a := range node.Actions {
				a.SendImmediate = &v
			}
		default:
			p.errorf("unexpected token %s in trigger block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Action Parsing ----------

func (p *Parser) parseAction() *ActionNode {
	pos := p.cur().Pos
	node := &ActionNode{NodeBase: NodeBase{Pos: pos}}

	switch p.cur().Kind {
	case TokNextScene:
		p.advance()
		node.ActionType = "next_scene"
	case TokPrevScene:
		p.advance()
		node.ActionType = "prev_scene"
	case TokJumpToEnactment:
		p.advance()
		node.ActionType = "jump_to_enactment"
		node.Target = p.expectString()
	case TokJumpToStoryline:
		p.advance()
		node.ActionType = "jump_to_storyline"
		node.Target = p.expectString()
	case TokAdvanceToNextStoryline:
		p.advance()
		node.ActionType = "advance_to_next_storyline"
		node.AdvanceToNextStoryline = true
	case TokEndStory:
		p.advance()
		node.ActionType = "end_story"
		node.EndStory = true
	case TokMarkComplete:
		p.advance()
		node.ActionType = "mark_complete"
		node.MarkComplete = true
	case TokMarkFailed:
		p.advance()
		node.ActionType = "mark_failed"
		node.MarkFailed = true
	case TokUnsubscribe:
		p.advance()
		node.ActionType = "unsubscribe"
		node.Unsubscribe = true
	case TokGiveBadge:
		p.advance()
		name := p.expectString()
		node.ActionType = "give_badge"
		node.BadgeTransaction = &BadgeTransactionNode{
			NodeBase:   NodeBase{Pos: pos},
			GiveBadges: []string{name},
		}
	case TokRemoveBadge:
		p.advance()
		name := p.expectString()
		node.ActionType = "remove_badge"
		node.BadgeTransaction = &BadgeTransactionNode{
			NodeBase:     NodeBase{Pos: pos},
			RemoveBadges: []string{name},
		}
	case TokRetryScene:
		p.advance()
		node.ActionType = "retry_scene"
		p.parseRetryModifiers(node)
	case TokRetryEnactment:
		p.advance()
		node.ActionType = "retry_enactment"
		p.parseRetryModifiers(node)
	case TokLoopToEnactment:
		p.advance()
		node.ActionType = "loop_to_enactment"
		node.Target = p.expectString()
		p.parseRetryModifiers(node)
	case TokLoopToStoryline:
		p.advance()
		node.ActionType = "loop_to_storyline"
		node.Target = p.expectString()
		p.parseRetryModifiers(node)
	case TokLoopToStartEnactment:
		p.advance()
		node.ActionType = "loop_to_start_enactment"
		p.parseRetryModifiers(node)
	case TokLoopToStartStoryline:
		p.advance()
		node.ActionType = "loop_to_start_storyline"
		p.parseRetryModifiers(node)
	case TokWait:
		p.advance()
		node.ActionType = "wait"
		node.Wait = p.parseDuration()
	case TokSendImmediate:
		p.advance()
		v := p.expectBool()
		node.ActionType = "send_immediate"
		node.SendImmediate = &v
	case TokNextEnactment:
		p.advance()
		node.ActionType = "next_enactment"
		node.Target = p.expectString()
	case TokJumpToStage:
		p.advance()
		node.ActionType = "jump_to_stage"
		node.Target = p.expectString()
	case TokStartStory:
		p.advance()
		node.ActionType = "start_story"
		node.Target = p.expectString()
	case TokSendEmailAction:
		p.advance()
		node.ActionType = "send_email"
		node.Target = p.expectString()
	case TokRedirect:
		p.advance()
		node.ActionType = "redirect"
		node.Target = p.expectString()
	case TokProvideDownload:
		p.advance()
		node.ActionType = "provide_download"
		node.Target = p.expectString()
	default:
		p.errorf("expected action keyword, got %s (%q)", p.cur().Kind, p.cur().Literal)
		p.advance()
		return nil
	}

	return node
}

func (p *Parser) parseRetryModifiers(node *ActionNode) {
	// up_to N
	if p.check(TokUpTo) {
		p.advance()
		v := p.expectInt()
		node.RetryMaxCount = &v
		// optional "times"
		if p.check(TokTimes) {
			p.advance()
		}
	}
	// else { ... } or else do <action>
	if p.check(TokElse) {
		p.advance()
		if p.check(TokLBrace) {
			p.expect(TokLBrace)
			for !p.check(TokRBrace) && !p.atEnd() {
				if p.check(TokDo) {
					p.advance()
				}
				action := p.parseAction()
				if action != nil {
					node.RetryFallback = append(node.RetryFallback, action)
				}
			}
			p.expect(TokRBrace)
		} else if p.check(TokDo) {
			p.advance()
			action := p.parseAction()
			if action != nil {
				node.RetryFallback = append(node.RetryFallback, action)
			}
		}
	}
}

// ---------- Condition Parsing ----------

func (p *Parser) parseCondition() *ConditionNode {
	pos := p.cur().Pos
	node := &ConditionNode{NodeBase: NodeBase{Pos: pos}}

	switch p.cur().Kind {
	case TokHasBadge:
		p.advance()
		node.ConditionType = "has_badge"
		node.Value = p.expectString()
	case TokNotHasBadge:
		p.advance()
		node.ConditionType = "not_has_badge"
		node.Value = p.expectString()
	case TokHasTag:
		p.advance()
		node.ConditionType = "has_tag"
		node.Value = p.expectString()
	case TokNotHasTag:
		p.advance()
		node.ConditionType = "not_has_tag"
		node.Value = p.expectString()
	case TokAnd:
		p.advance()
		node.ConditionType = "and"
		if p.check(TokLBrace) {
			p.expect(TokLBrace)
			for !p.check(TokRBrace) && !p.atEnd() {
				child := p.parseCondition()
				if child != nil {
					node.Children = append(node.Children, child)
				}
			}
			p.expect(TokRBrace)
		}
	case TokOr:
		p.advance()
		node.ConditionType = "or"
		if p.check(TokLBrace) {
			p.expect(TokLBrace)
			for !p.check(TokRBrace) && !p.atEnd() {
				child := p.parseCondition()
				if child != nil {
					node.Children = append(node.Children, child)
				}
			}
			p.expect(TokRBrace)
		}
	case TokNot:
		p.advance()
		node.ConditionType = "not"
		child := p.parseCondition()
		if child != nil {
			node.Children = append(node.Children, child)
		}
	default:
		p.errorf("expected condition keyword, got %s", p.cur().Kind)
		p.advance()
		return nil
	}

	// Check for chained and/or
	if p.check(TokAnd) && node.ConditionType != "and" {
		wrapper := &ConditionNode{
			NodeBase:      NodeBase{Pos: pos},
			ConditionType: "and",
			Children:      []*ConditionNode{node},
		}
		p.advance()
		right := p.parseCondition()
		if right != nil {
			wrapper.Children = append(wrapper.Children, right)
		}
		return wrapper
	}
	if p.check(TokOr) && node.ConditionType != "or" {
		wrapper := &ConditionNode{
			NodeBase:      NodeBase{Pos: pos},
			ConditionType: "or",
			Children:      []*ConditionNode{node},
		}
		p.advance()
		right := p.parseCondition()
		if right != nil {
			wrapper.Children = append(wrapper.Children, right)
		}
		return wrapper
	}

	return node
}

// ---------- Duration Parsing ----------

func (p *Parser) parseDuration() *DurationNode {
	pos := p.cur().Pos

	// Duration can be: TokDuration (e.g. "1d"), or TokInt followed by a time unit keyword
	if p.check(TokDuration) {
		tok := p.advance()
		amount, unit := parseDurationLiteral(tok.Literal)
		return &DurationNode{
			NodeBase: NodeBase{Pos: pos},
			Amount:   amount,
			Unit:     unit,
			RawValue: tok.Literal,
		}
	}

	if p.check(TokInt) {
		amount := p.expectInt()
		// expect a time unit word
		unitTok := p.cur()
		unit := ""
		switch strings.ToLower(unitTok.Literal) {
		case "day", "days", "d":
			unit = "d"
			p.advance()
		case "hour", "hours", "h":
			unit = "h"
			p.advance()
		case "minute", "minutes", "min", "m":
			unit = "m"
			p.advance()
		case "second", "seconds", "sec", "s":
			unit = "s"
			p.advance()
		default:
			p.errorf("expected time unit (day/hour/minute/second), got %q", unitTok.Literal)
		}
		return &DurationNode{
			NodeBase: NodeBase{Pos: pos},
			Amount:   amount,
			Unit:     unit,
			RawValue: fmt.Sprintf("%d%s", amount, unit),
		}
	}

	// Try string literal like "1 day"
	if p.check(TokString) {
		raw := p.expectString()
		amount, unit := parseDurationString(raw)
		return &DurationNode{
			NodeBase: NodeBase{Pos: pos},
			Amount:   amount,
			Unit:     unit,
			RawValue: raw,
		}
	}

	p.errorf("expected duration, got %s", p.cur().Kind)
	return &DurationNode{NodeBase: NodeBase{Pos: pos}}
}

// parseDurationLiteral parses "1d", "2h", "30m" etc.
func parseDurationLiteral(s string) (int, string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, ""
	}
	amount, _ := strconv.Atoi(s[:i])
	unit := s[i:]
	return amount, unit
}

// parseDurationString parses strings like "1 day", "2 hours".
func parseDurationString(s string) (int, string) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		// try without space: "1d"
		return parseDurationLiteral(s)
	}
	amount, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}
	switch strings.ToLower(parts[1]) {
	case "day", "days", "d":
		return amount, "d"
	case "hour", "hours", "h":
		return amount, "h"
	case "minute", "minutes", "min", "m":
		return amount, "m"
	case "second", "seconds", "sec", "s":
		return amount, "s"
	}
	return amount, parts[1]
}

// ---------- Lifecycle Block Parsing ----------

func (p *Parser) parseLifecycleBlock() *LifecycleBlock {
	pos := p.cur().Pos
	p.advance() // consume on_begin
	node := &LifecycleBlock{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokGiveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.GiveBadges = append(node.BadgeTransaction.GiveBadges, name)
		case TokRemoveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.RemoveBadges = append(node.BadgeTransaction.RemoveBadges, name)
		default:
			p.errorf("unexpected token %s in on_begin block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

func (p *Parser) parseOnCompleteBlock() *OnCompleteBlock {
	pos := p.cur().Pos
	p.advance() // consume on_complete
	node := &OnCompleteBlock{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokGiveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.GiveBadges = append(node.BadgeTransaction.GiveBadges, name)
		case TokRemoveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.RemoveBadges = append(node.BadgeTransaction.RemoveBadges, name)
		case TokNextStory:
			p.advance()
			node.NextStory = p.expectString()
		default:
			p.errorf("unexpected token %s in on_complete block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

func (p *Parser) parseOnFailBlock() *OnFailBlock {
	pos := p.cur().Pos
	p.advance() // consume on_fail
	node := &OnFailBlock{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokGiveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.GiveBadges = append(node.BadgeTransaction.GiveBadges, name)
		case TokRemoveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.RemoveBadges = append(node.BadgeTransaction.RemoveBadges, name)
		case TokNextStory:
			p.advance()
			node.NextStory = p.expectString()
		default:
			p.errorf("unexpected token %s in on_fail block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

func (p *Parser) parseStorylineCompleteBlock() *StorylineCompleteBlock {
	pos := p.cur().Pos
	p.advance() // consume on_complete
	node := &StorylineCompleteBlock{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokGiveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.GiveBadges = append(node.BadgeTransaction.GiveBadges, name)
		case TokRemoveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.RemoveBadges = append(node.BadgeTransaction.RemoveBadges, name)
		case TokNextStoryline:
			p.advance()
			node.NextStoryline = p.expectString()
		case TokConditionalRoute, TokRoute:
			cr := p.parseConditionalRoute()
			if cr != nil {
				node.ConditionalRoutes = append(node.ConditionalRoutes, cr)
			}
		default:
			p.errorf("unexpected token %s in storyline on_complete block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

func (p *Parser) parseStorylineFailBlock() *StorylineFailBlock {
	pos := p.cur().Pos
	p.advance() // consume on_fail
	node := &StorylineFailBlock{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokGiveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.GiveBadges = append(node.BadgeTransaction.GiveBadges, name)
		case TokRemoveBadge:
			p.advance()
			name := p.expectString()
			if node.BadgeTransaction == nil {
				node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.BadgeTransaction.RemoveBadges = append(node.BadgeTransaction.RemoveBadges, name)
		case TokNextStoryline:
			p.advance()
			node.NextStoryline = p.expectString()
		case TokConditionalRoute, TokRoute:
			cr := p.parseConditionalRoute()
			if cr != nil {
				node.ConditionalRoutes = append(node.ConditionalRoutes, cr)
			}
		default:
			p.errorf("unexpected token %s in storyline on_fail block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Conditional Route Parsing ----------

func (p *Parser) parseConditionalRoute() *ConditionalRouteNode {
	pos := p.cur().Pos
	p.advance() // consume route or conditional_route
	node := &ConditionalRouteNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokRequiredBadges:
			node.RequiredBadges = p.parseRequiredBadges()
		case TokNextStoryline:
			p.advance()
			node.NextStoryline = p.expectString()
		case TokPriority:
			p.advance()
			v := p.expectInt()
			node.Priority = &v
		default:
			p.errorf("unexpected token %s in conditional_route block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Required Badges Parsing ----------

func (p *Parser) parseRequiredBadges() *RequiredBadgesNode {
	pos := p.cur().Pos
	p.advance() // consume required_badges
	node := &RequiredBadgesNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokMustHave:
			p.advance()
			node.MustHave = append(node.MustHave, p.parseStringListOrSingle()...)
		case TokMustNotHave:
			p.advance()
			node.MustNotHave = append(node.MustNotHave, p.parseStringListOrSingle()...)
		default:
			p.errorf("unexpected token %s in required_badges block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- String List Parsing ----------

func (p *Parser) parseStringList() []string {
	var list []string
	p.expect(TokLBracket)
	for !p.check(TokRBracket) && !p.atEnd() {
		list = append(list, p.expectString())
		if p.check(TokComma) {
			p.advance()
		}
	}
	p.expect(TokRBracket)
	return list
}

func (p *Parser) parseStringListOrSingle() []string {
	if p.check(TokLBracket) {
		return p.parseStringList()
	}
	return []string{p.expectString()}
}

func (p *Parser) parseVarsMap() map[string]string {
	m := make(map[string]string)
	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		key := ""
		if p.check(TokString) {
			key = p.expectString()
		} else if p.check(TokIdent) {
			key = p.advance().Literal
		} else {
			p.errorf("expected key in vars map, got %s", p.cur().Kind)
			p.advance()
			continue
		}
		if p.check(TokColon) || p.check(TokEquals) {
			p.advance()
		}
		val := p.expectString()
		m[key] = val
		if p.check(TokComma) {
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return m
}

// ---------- Funnel Parsing ----------

// parseFunnel parses: funnel "name" { domain "..." ai context ... route "name" { ... } }
func (p *Parser) parseFunnel() *FunnelNode {
	pos := p.cur().Pos
	p.expect(TokFunnel)
	name := p.expectString()
	node := &FunnelNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokDomain:
			p.advance()
			node.Domain = p.expectString()
		case TokAI:
			node.AIContext = p.parseAIContext()
		case TokRoute:
			r := p.parseRoute()
			if r != nil {
				node.Routes = append(node.Routes, r)
			}
		default:
			p.errorf("unexpected token %s in funnel block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseRoute parses: route "name" { order N must_have_badge/must_not_have_badge stage { ... } }
func (p *Parser) parseRoute() *RouteNode {
	pos := p.cur().Pos
	p.expect(TokRoute)
	name := p.expectString()
	node := &RouteNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOrder:
			p.advance()
			v := p.expectInt()
			node.Order = &v
		case TokRequiredBadges:
			node.RequiredBadges = p.parseRequiredBadges()
		case TokMustHaveBadge:
			p.advance()
			badgeName := p.expectString()
			if node.RequiredBadges == nil {
				node.RequiredBadges = &RequiredBadgesNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.RequiredBadges.MustHave = append(node.RequiredBadges.MustHave, badgeName)
		case TokMustNotHaveBadge:
			p.advance()
			badgeName := p.expectString()
			if node.RequiredBadges == nil {
				node.RequiredBadges = &RequiredBadgesNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.RequiredBadges.MustNotHave = append(node.RequiredBadges.MustNotHave, badgeName)
		case TokStage:
			s := p.parseStage()
			if s != nil {
				node.Stages = append(node.Stages, s)
			}
		default:
			p.errorf("unexpected token %s in route block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseStage parses: stage "name" { path "..." page { ... } on submit/abandon/purchase { ... } pdf ... }
func (p *Parser) parseStage() *StageNode {
	pos := p.cur().Pos
	p.expect(TokStage)
	name := p.expectString()
	node := &StageNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOrder:
			p.advance()
			v := p.expectInt()
			node.Order = &v
		case TokPath:
			p.advance()
			node.Path = p.expectString()
		case TokPage:
			pg := p.parsePage()
			if pg != nil {
				node.Pages = append(node.Pages, pg)
			}
		case TokOn:
			tr := p.parseTrigger()
			if tr != nil {
				node.Triggers = append(node.Triggers, tr)
			}
		case TokPdf:
			node.PDFConfig = p.parsePDFConfig()
		case TokLeadMagnet:
			lm := p.parseLeadMagnet()
			if lm != nil {
				node.LeadMagnets = append(node.LeadMagnets, lm)
			}
		default:
			p.errorf("unexpected token %s in stage block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseLeadMagnet parses:
//
//	lead_magnet {
//	    type "worksheet"
//	    reference "https://..."
//	    context "Generate a 5-page worksheet on..."
//	    theme "minimal"
//	}
func (p *Parser) parseLeadMagnet() *LeadMagnetNode {
	pos := p.cur().Pos
	p.expect(TokLeadMagnet)
	node := &LeadMagnetNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokType:
			p.advance()
			if p.check(TokString) {
				node.AssetType = p.advance().Literal
			} else {
				node.AssetType = p.advanceIdentLike()
			}
		case TokReference:
			p.advance()
			ref := ""
			if p.check(TokString) {
				ref = p.advance().Literal
			} else {
				ref = p.advanceIdentLike()
			}
			if ref != "" {
				node.References = append(node.References, ref)
			}
		case TokContext:
			p.advance()
			if p.check(TokString) {
				node.Instruction = p.advance().Literal
			} else {
				node.Instruction = p.advanceIdentLike()
			}
		case TokTheme:
			p.advance()
			if p.check(TokString) {
				node.Theme = p.advance().Literal
			} else {
				node.Theme = p.advanceIdentLike()
			}
		default:
			p.errorf("unexpected token %s in lead_magnet block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parsePage parses: page "name" { template "..." ai context ... block "id" { ... } form "name" { ... } }
func (p *Parser) parsePage() *PageNode {
	pos := p.cur().Pos
	p.expect(TokPage)
	name := p.expectString()
	node := &PageNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTemplate:
			p.advance()
			node.TemplateName = p.expectString()
		case TokAI:
			node.AIContext = p.parseAIContext()
		case TokBlock:
			b := p.parseBlock()
			if b != nil {
				node.Blocks = append(node.Blocks, b)
			}
		case TokForm:
			f := p.parseForm()
			if f != nil {
				node.Forms = append(node.Forms, f)
			}
		case TokLeadMagnet:
			lm := p.parseLeadMagnet()
			if lm != nil {
				node.LeadMagnets = append(node.LeadMagnets, lm)
			}
		default:
			p.errorf("unexpected token %s in page block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseBlock parses: block "section_id" { length short/medium/long ai context ... prompt "..." }
func (p *Parser) parseBlock() *BlockNode {
	pos := p.cur().Pos
	p.expect(TokBlock)
	sectionID := p.expectString()
	node := &BlockNode{NodeBase: NodeBase{Pos: pos}, SectionID: sectionID}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokType:
			p.advance()
			if p.check(TokVideo) {
				node.BlockType = p.advance().Literal
			} else if p.check(TokString) {
				node.BlockType = p.expectString()
			} else if p.isIdentLike() {
				node.BlockType = p.advanceIdentLike()
			} else {
				p.errorf("expected block type value, got %s", p.cur().Kind)
				p.advance()
			}
		case TokSourceURL:
			p.advance()
			node.SourceURL = p.expectString()
		case TokAutoplay:
			p.advance()
			if p.check(TokBool) {
				node.Autoplay = p.cur().Literal == "true"
				p.advance()
			} else if p.check(TokTrue) {
				p.advance()
				node.Autoplay = true
			} else if p.check(TokFalse) {
				p.advance()
				node.Autoplay = false
			} else {
				node.Autoplay = true
			}
		case TokLength:
			p.advance()
			if node.ContentGen == nil {
				node.ContentGen = &ContentGenNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			if p.check(TokString) {
				node.ContentGen.Length = p.expectString()
			} else if p.isIdentLike() {
				node.ContentGen.Length = p.advanceIdentLike()
			} else {
				p.errorf("expected length value, got %s", p.cur().Kind)
				p.advance()
			}
		case TokPrompt:
			p.advance()
			if node.ContentGen == nil {
				node.ContentGen = &ContentGenNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			node.ContentGen.PromptAppend = p.expectString()
		case TokContext:
			// context "url" inside a block — adds to ContentGen.ContextURLs
			p.advance()
			if node.ContentGen == nil {
				node.ContentGen = &ContentGenNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
			}
			if p.check(TokString) {
				node.ContentGen.ContextURLs = append(node.ContentGen.ContextURLs, p.advance().Literal)
			} else if p.isIdentLike() {
				node.ContentGen.ContextURLs = append(node.ContentGen.ContextURLs, p.advanceIdentLike())
			} else {
				p.errorf("expected context URL string, got %s", p.cur().Kind)
				p.advance()
			}
		case TokIdent:
			// Handle known bare-identifier properties like section_id
			if p.cur().Literal == "section_id" {
				p.advance() // consume "section_id"
				if p.check(TokString) {
					node.SectionID = p.advance().Literal
				} else if p.isIdentLike() {
					node.SectionID = p.advanceIdentLike()
				}
			} else {
				p.errorf("unexpected token %s in block", p.cur().Kind)
				p.advance()
			}
		case TokAI:
			node.AIContext = p.parseAIContext()
		case TokMediaRef:
			p.advance()
			node.MediaPublicId = p.expectString()
		case TokPlayerPreset:
			p.advance()
			node.PlayerPresetId = p.expectString()
		default:
			p.errorf("unexpected token %s in block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseForm parses: form "name" { type checkout/lead_capture/upsell product_id "..." field email required ... }
func (p *Parser) parseForm() *FormNode {
	pos := p.cur().Pos
	p.expect(TokForm)
	name := p.expectString()
	node := &FormNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokType:
			p.advance()
			if p.check(TokString) {
				node.FormType = p.expectString()
			} else if p.check(TokCheckout) {
				node.FormType = "checkout"
				p.advance()
			} else if p.check(TokLeadCapture) {
				node.FormType = "lead_capture"
				p.advance()
			} else if p.check(TokUpsell) {
				node.FormType = "upsell"
				p.advance()
			} else if p.check(TokOneClickUpsell) {
				node.FormType = "one_click_upsell"
				p.advance()
			} else if p.isIdentLike() {
				node.FormType = p.advanceIdentLike()
			} else {
				p.errorf("expected form type, got %s", p.cur().Kind)
				p.advance()
			}
		case TokProductId:
			p.advance()
			node.ProductID = p.expectString()
		case TokOffer:
			p.advance()
			node.OfferID = p.expectString()
		case TokOrderBump:
			ob := p.parseOrderBump()
			if ob != nil {
				node.OrderBumps = append(node.OrderBumps, ob)
			}
		case TokField:
			ff := p.parseFormField()
			if ff != nil {
				node.Fields = append(node.Fields, ff)
			}
		default:
			p.errorf("unexpected token %s in form block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseFormField parses: field email [required] or field first_name
func (p *Parser) parseFormField() *FormFieldNode {
	pos := p.cur().Pos
	p.expect(TokField)
	node := &FormFieldNode{NodeBase: NodeBase{Pos: pos}}

	// Field name (can be identifier or string)
	if p.check(TokString) {
		node.FieldName = p.expectString()
	} else if p.isIdentLike() {
		node.FieldName = p.advanceIdentLike()
	} else if p.check(TokEmail) {
		node.FieldName = "email"
		p.advance()
	} else {
		p.errorf("expected field name, got %s", p.cur().Kind)
		p.advance()
		return nil
	}

	// Optional field type
	if p.check(TokCustom) {
		node.FieldType = "custom"
		p.advance()
		// Custom field name follows
		if p.check(TokString) {
			node.CustomField = p.expectString()
		} else if p.isIdentLike() {
			node.CustomField = p.advanceIdentLike()
		}
		if node.CustomField != "" && node.FieldName == node.CustomField {
			// field name becomes the custom field name
		} else if node.CustomField != "" {
			node.FieldName = node.CustomField
		}
	} else if p.check(TokString) {
		node.FieldType = p.expectString()
	} else if p.check(TokEmail) {
		node.FieldType = "email"
		p.advance()
	}

	// If no explicit type set, derive from name
	if node.FieldType == "" {
		node.FieldType = node.FieldName
	}

	// Optional "required"
	if p.check(TokRequired) {
		p.advance()
		node.Required = true
	}

	return node
}

// parseAIContext parses: ai context global "url" "ref" or ai context extend "ref"
func (p *Parser) parseAIContext() *AIContextNode {
	pos := p.cur().Pos
	p.expect(TokAI)
	p.expect(TokContext)
	node := &AIContextNode{NodeBase: NodeBase{Pos: pos}}

	// Mode: global or extend
	if p.check(TokGlobal) {
		node.Mode = "global"
		p.advance()
	} else if p.check(TokExtend) {
		node.Mode = "extend"
		p.advance()
	}

	// Collect URLs and refs (strings and plain identifiers only).
	// Stop at structural keywords (route, stage, page, block, form, on, etc.)
	// to avoid consuming tokens that belong to the parent parser context.
	for p.check(TokString) || p.check(TokIdent) {
		if p.check(TokString) {
			val := p.expectString()
			if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
				node.ContextURLs = append(node.ContextURLs, val)
			} else {
				node.ContextRefs = append(node.ContextRefs, val)
			}
		} else {
			node.ContextRefs = append(node.ContextRefs, p.advance().Literal)
		}
	}

	return node
}

// parsePDFConfig parses: pdf new ai context global "url" "ref"
func (p *Parser) parsePDFConfig() *PDFConfigNode {
	pos := p.cur().Pos
	p.expect(TokPdf)
	node := &PDFConfigNode{NodeBase: NodeBase{Pos: pos}}

	if p.check(TokNew) {
		p.advance()
		node.AIGenerated = true
	}

	if p.check(TokAI) {
		node.AIContext = p.parseAIContext()
	}

	return node
}

// ---------- E-Commerce Parsing ----------

// parseProductDecl parses: product "Name" { type "course" description "..." module "M1" { lesson "L1" { ... } } }
func (p *Parser) parseProductDecl() *ProductDeclNode {
	pos := p.cur().Pos
	p.expect(TokProduct)
	name := p.expectString()
	node := &ProductDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokType:
			p.advance()
			if p.check(TokString) {
				node.ProductType = p.expectString()
			} else if p.isIdentLike() {
				node.ProductType = p.advanceIdentLike()
			} else {
				p.errorf("expected product type, got %s", p.cur().Kind)
				p.advance()
			}
		case TokDescription:
			p.advance()
			node.Description = p.expectString()
		case TokModule:
			m := p.parseModule()
			if m != nil {
				node.Modules = append(node.Modules, m)
			}
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokInstructor:
			p.advance()
			node.Instructor = p.expectString()
		case TokTheme:
			p.advance()
			if p.check(TokString) {
				node.ThumbnailURL = p.expectString()
			} else if p.isIdentLike() {
				node.ThumbnailURL = p.advanceIdentLike()
			}
		case TokDescriptionGen:
			node.DescriptionGen = p.parseDescriptionGen()
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			switch ident {
			case "description":
				node.Description = p.expectString()
			case "thumbnail":
				if p.check(TokString) {
					node.ThumbnailURL = p.expectString()
				} else if p.isIdentLike() {
					node.ThumbnailURL = p.advanceIdentLike()
				}
			case "status":
				if p.check(TokString) {
					node.Status = p.expectString()
				} else if p.isIdentLike() {
					node.Status = p.advanceIdentLike()
				}
			case "instructor":
				node.Instructor = p.expectString()
			case "title":
				node.Title = p.expectString()
			default:
				p.errorf("unexpected identifier %q in product block", ident)
			}
		default:
			p.errorf("unexpected token %s in product block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseCourseDecl parses: course "Name" { description "..." instructor "..." module "..." { ... } ... }
func (p *Parser) parseCourseDecl() *CourseDeclNode {
	pos := p.cur().Pos
	p.expect(TokCourse)
	name := p.expectString()
	node := &CourseDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokDescription:
			p.advance()
			node.Description = p.expectString()
		case TokInstructor:
			p.advance()
			node.Instructor = p.expectString()
		case TokModule:
			m := p.parseModule()
			if m != nil {
				node.Modules = append(node.Modules, m)
			}
		case TokDescriptionGen:
			node.DescriptionGen = p.parseDescriptionGen()
		case TokCertificate:
			node.CertConfig = p.parseCourseCertConfig()
		case TokReference:
			p.advance()
			node.References = append(node.References, p.expectString())
		case TokAudience:
			p.advance()
			node.Audience = p.expectString()
		case TokOutcome:
			p.advance()
			node.Outcome = p.expectString()
		case TokTone:
			p.advance()
			node.Tone = p.expectString()
		case TokDefaultMedia:
			p.advance()
			if p.check(TokString) {
				node.DefaultMedia = p.expectString()
			} else if p.isIdentLike() {
				node.DefaultMedia = p.advanceIdentLike()
			}
		case TokGenMode:
			p.advance()
			if p.check(TokString) {
				node.Mode = p.expectString()
			} else if p.isIdentLike() {
				node.Mode = p.advanceIdentLike()
			}
		case TokTheme:
			p.advance()
			if p.check(TokString) {
				node.ThumbnailURL = p.expectString()
			} else if p.isIdentLike() {
				node.ThumbnailURL = p.advanceIdentLike()
			}
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			switch ident {
			case "description":
				node.Description = p.expectString()
			case "instructor":
				node.Instructor = p.expectString()
			case "thumbnail":
				if p.check(TokString) {
					node.ThumbnailURL = p.expectString()
				} else if p.isIdentLike() {
					node.ThumbnailURL = p.advanceIdentLike()
				}
			case "status":
				if p.check(TokString) {
					node.Status = p.expectString()
				} else if p.isIdentLike() {
					node.Status = p.advanceIdentLike()
				}
			case "audience":
				node.Audience = p.expectString()
			case "outcome":
				node.Outcome = p.expectString()
			case "tone":
				node.Tone = p.expectString()
			case "extra_context":
				node.ExtraContext = p.expectString()
			case "default_media":
				if p.check(TokString) {
					node.DefaultMedia = p.expectString()
				} else if p.isIdentLike() {
					node.DefaultMedia = p.advanceIdentLike()
				}
			case "mode":
				if p.check(TokString) {
					node.Mode = p.expectString()
				} else if p.isIdentLike() {
					node.Mode = p.advanceIdentLike()
				}
			default:
				p.errorf("unexpected identifier %q in course block", ident)
			}
		default:
			p.errorf("unexpected token %s in course block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseCourseCertConfig parses: certificate { enabled true template_name "default" ... }
func (p *Parser) parseCourseCertConfig() *CourseCertConfigNode {
	pos := p.cur().Pos
	p.expect(TokCertificate)
	node := &CourseCertConfigNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokEnabled:
			p.advance()
			if p.check(TokBool) || p.check(TokTrue) || p.check(TokFalse) {
				node.Enabled = p.cur().Literal == "true"
				p.advance()
			} else {
				node.Enabled = true
			}
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			switch ident {
			case "template_name":
				node.TemplateName = p.expectString()
			case "title":
				node.Title = p.expectString()
			case "logo_url":
				node.LogoURL = p.expectString()
			case "accent_color":
				node.AccentColor = p.expectString()
			case "enabled":
				if p.check(TokBool) || p.check(TokTrue) || p.check(TokFalse) {
					node.Enabled = p.cur().Literal == "true"
					p.advance()
				} else {
					node.Enabled = true
				}
			default:
				p.errorf("unexpected identifier %q in certificate config", ident)
			}
		default:
			p.errorf("unexpected token %s in certificate config", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseOfferDecl parses: offer "Name" { pricing_model one_time price 497.00 currency "usd" includes_product "..." grants_badge "..." on purchase { ... } }
func (p *Parser) parseOfferDecl() *OfferDeclNode {
	pos := p.cur().Pos
	p.expect(TokOffer)
	name := p.expectString()
	node := &OfferDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name, Currency: "usd"}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokPricingModel:
			p.advance()
			if p.check(TokString) {
				node.PricingModel = p.expectString()
			} else if p.isIdentLike() {
				node.PricingModel = p.advanceIdentLike()
			} else {
				p.errorf("expected pricing model, got %s", p.cur().Kind)
				p.advance()
			}
		case TokPrice:
			p.advance()
			if p.check(TokFloat) {
				node.Price = p.expectFloat()
			} else if p.check(TokInt) {
				node.Price = float64(p.expectInt())
			} else {
				p.errorf("expected price value, got %s", p.cur().Kind)
				p.advance()
			}
		case TokCurrency:
			p.advance()
			node.Currency = p.expectString()
		case TokIncludesProduct:
			p.advance()
			node.IncludedProducts = append(node.IncludedProducts, p.expectString())
		case TokGrantsBadge:
			p.advance()
			node.GrantedBadges = append(node.GrantedBadges, p.expectString())
		case TokOn:
			p.advance()
			if p.check(TokPurchase) {
				p.advance()
				node.OnPurchase = p.parseOfferLifecycleBlock()
			} else {
				p.errorf("expected 'purchase' after 'on' in offer block, got %s", p.cur().Kind)
				p.advance()
			}
		default:
			p.errorf("unexpected token %s in offer block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseOrderBump parses: order_bump "Name" { offer "..." text "..." }
func (p *Parser) parseOrderBump() *OrderBumpNode {
	pos := p.cur().Pos
	p.expect(TokOrderBump)
	name := p.expectString()
	node := &OrderBumpNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokOffer:
			p.advance()
			node.OfferID = p.expectString()
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			if ident == "text" {
				node.Text = p.expectString()
			} else {
				p.errorf("unexpected identifier %q in order_bump block", ident)
			}
		default:
			p.errorf("unexpected token %s in order_bump block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// ---------- Quiz Parsing ----------

// parseQuiz parses: quiz "Name" { question "..." { answer "..." add_score N ... } on complete { ... } }
func (p *Parser) parseQuiz() *QuizNode {
	pos := p.cur().Pos
	p.expect(TokQuiz)
	name := p.expectString()
	node := &QuizNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokQuestion:
			q := p.parseQuestion()
			if q != nil {
				node.Questions = append(node.Questions, q)
			}
		case TokOn:
			p.advance()
			if p.check(TokOnComplete) || (p.isIdentLike() && p.cur().Literal == "complete") {
				p.advance()
				node.OnComplete = p.parseQuizOnComplete()
			} else {
				p.errorf("expected 'complete' after 'on' in quiz block, got %s", p.cur().Kind)
				p.advance()
			}
		default:
			p.errorf("unexpected token %s in quiz block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseQuestion parses: question "text" { answer "text" add_score N ... }
func (p *Parser) parseQuestion() *QuestionNode {
	pos := p.cur().Pos
	p.expect(TokQuestion)
	text := p.expectString()
	node := &QuestionNode{NodeBase: NodeBase{Pos: pos}, Text: text}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokAnswer:
			a := p.parseAnswer()
			if a != nil {
				node.Answers = append(node.Answers, a)
			}
		default:
			p.errorf("unexpected token %s in question block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseAnswer parses: answer "text" add_score N
func (p *Parser) parseAnswer() *AnswerNode {
	pos := p.cur().Pos
	p.expect(TokAnswer)
	text := p.expectString()
	node := &AnswerNode{NodeBase: NodeBase{Pos: pos}, Text: text}

	if p.check(TokAddScore) {
		p.advance()
		node.AddScore = p.expectInt()
	}

	return node
}

// parseQuizOnComplete parses: { if score > N { do give_badge "..." } else { ... } }
func (p *Parser) parseQuizOnComplete() *QuizOnCompleteNode {
	node := &QuizOnCompleteNode{NodeBase: NodeBase{Pos: p.cur().Pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		if p.check(TokIf) || p.check(TokElseIf) || p.check(TokElse) {
			rule := p.parseScoreRule()
			if rule != nil {
				node.ScoreRules = append(node.ScoreRules, rule)
			}
		} else {
			p.errorf("unexpected token %s in quiz on_complete block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseScoreRule parses: if score > N { do give_badge "..." } or else { ... }
func (p *Parser) parseScoreRule() *ScoreRuleNode {
	node := &ScoreRuleNode{NodeBase: NodeBase{Pos: p.cur().Pos}}

	if p.check(TokElse) {
		p.advance()
		node.Operator = "else"
	} else {
		p.advance() // consume if/else_if
		if p.check(TokScore) {
			p.advance()
		}
		// Parse comparison operator
		switch p.cur().Kind {
		case TokGT:
			node.Operator = ">"
		case TokLT:
			node.Operator = "<"
		case TokGTEQ:
			node.Operator = ">="
		case TokLTEQ:
			node.Operator = "<="
		case TokEQEQ:
			node.Operator = "=="
		default:
			p.errorf("expected comparison operator, got %s", p.cur().Kind)
		}
		p.advance()
		node.Threshold = p.expectInt()
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		if p.check(TokDo) {
			p.advance()
			action := p.parseActionNode()
			if action != nil {
				node.Actions = append(node.Actions, action)
			}
		} else {
			p.errorf("unexpected token %s in score rule block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseOfferLifecycleBlock parses: { do give_badge "..." do start_story "..." }
// Used for offer on_purchase blocks which use `do` prefix syntax.
func (p *Parser) parseOfferLifecycleBlock() *LifecycleBlock {
	node := &LifecycleBlock{NodeBase: NodeBase{Pos: p.cur().Pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokDo:
			p.advance()
			switch p.cur().Kind {
			case TokGiveBadge:
				p.advance()
				name := p.expectString()
				if node.BadgeTransaction == nil {
					node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
				}
				node.BadgeTransaction.GiveBadges = append(node.BadgeTransaction.GiveBadges, name)
			case TokRemoveBadge:
				p.advance()
				name := p.expectString()
				if node.BadgeTransaction == nil {
					node.BadgeTransaction = &BadgeTransactionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
				}
				node.BadgeTransaction.RemoveBadges = append(node.BadgeTransaction.RemoveBadges, name)
			case TokStartStory:
				p.advance()
				_ = p.expectString() // consume but lifecycle blocks don't have start_story - just skip
			default:
				p.errorf("unexpected action %s in offer on_purchase", p.cur().Kind)
				p.advance()
			}
		default:
			p.errorf("expected 'do' in offer on_purchase block, got %s", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseActionNode parses a single action like give_badge "name" or start_story "name".
func (p *Parser) parseActionNode() *ActionNode {
	node := &ActionNode{NodeBase: NodeBase{Pos: p.cur().Pos}}
	switch p.cur().Kind {
	case TokGiveBadge:
		p.advance()
		badgeName := p.expectString()
		node.BadgeTransaction = &BadgeTransactionNode{
			GiveBadges: []string{badgeName},
		}
	case TokRemoveBadge:
		p.advance()
		badgeName := p.expectString()
		node.BadgeTransaction = &BadgeTransactionNode{
			RemoveBadges: []string{badgeName},
		}
	case TokStartStory:
		p.advance()
		node.ActionType = "start_story"
		node.Target = p.expectString()
	case TokJumpToStage:
		p.advance()
		node.ActionType = "jump_to_stage"
		node.Target = p.expectString()
	default:
		p.errorf("unexpected action %s", p.cur().Kind)
		p.advance()
		return nil
	}
	return node
}

// ---------- Website / Site Parsing ----------

// parseSite parses: site "name" { domain "..." theme "..." seo { ... } navigation { ... } page "name" { ... } }
func (p *Parser) parseSite() *SiteNode {
	pos := p.cur().Pos
	p.expect(TokSite)
	name := p.expectString()
	node := &SiteNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokDomain:
			p.advance()
			node.Domain = p.expectString()
		case TokTheme:
			p.advance()
			node.Theme = p.expectString()
		case TokSEO:
			node.SEO = p.parseSEOBlock()
		case TokNavigation:
			node.Navigation = p.parseNavigationBlock()
		case TokPage:
			pg := p.parsePage()
			if pg != nil {
				node.Pages = append(node.Pages, pg)
			}
		default:
			p.errorf("unexpected token %s in site block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseSEOBlock parses: seo { title "..." description "..." }
func (p *Parser) parseSEOBlock() *SEONode {
	pos := p.cur().Pos
	p.expect(TokSEO)
	node := &SEONode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTitle:
			p.advance()
			node.MetaTitle = p.expectString()
		case TokDescription:
			p.advance()
			node.MetaDescription = p.expectString()
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			if ident == "og_image" || ident == "image" {
				node.OpenGraphImageURL = p.expectString()
			} else {
				p.errorf("unexpected identifier %q in seo block", ident)
			}
		default:
			p.errorf("unexpected token %s in seo block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseNavigationBlock parses: navigation { header { "Label" = "/path", ... } footer { ... } }
func (p *Parser) parseNavigationBlock() *NavigationNode {
	pos := p.cur().Pos
	p.expect(TokNavigation)
	node := &NavigationNode{
		NodeBase:    NodeBase{Pos: pos},
		HeaderLinks: make(map[string]string),
		FooterLinks: make(map[string]string),
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokHeader:
			p.advance()
			p.expect(TokLBrace)
			p.parseNavLinks(node.HeaderLinks)
			p.expect(TokRBrace)
		case TokFooter:
			p.advance()
			p.expect(TokLBrace)
			p.parseNavLinks(node.FooterLinks)
			p.expect(TokRBrace)
		default:
			p.errorf("unexpected token %s in navigation block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseNavLinks parses: "Label" = "/path", "Label2" = "/path2", ...
func (p *Parser) parseNavLinks(links map[string]string) {
	for !p.check(TokRBrace) && !p.atEnd() {
		label := p.expectString()
		p.expect(TokEquals)
		path := p.expectString()
		links[label] = path
		// Optional comma separator
		if p.check(TokComma) {
			p.advance()
		}
	}
}

// ---------- LMS Parsing ----------

// parseModule parses: module "Title" { lesson "Title" { ... } quiz "Title" { ... } }
func (p *Parser) parseModule() *ModuleNode {
	pos := p.cur().Pos
	p.expect(TokModule)
	title := p.expectString()
	node := &ModuleNode{NodeBase: NodeBase{Pos: pos}, Title: title}

	// Optional slug as second string
	if p.check(TokString) {
		node.Slug = p.expectString()
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokOrder:
			p.advance()
			node.Order = p.expectInt()
		case TokLesson:
			l := p.parseLesson()
			if l != nil {
				node.Lessons = append(node.Lessons, l)
			}
		case TokQuiz:
			q := p.parseLMSQuiz()
			if q != nil {
				node.Quizzes = append(node.Quizzes, q)
			}
		default:
			p.errorf("unexpected token %s in module block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseLesson parses: lesson "Title" { video_url "..." content "..." is_free true drip_days 7 duration "01:30:00" ... }
func (p *Parser) parseLesson() *LessonNode {
	pos := p.cur().Pos
	p.expect(TokLesson)
	title := p.expectString()
	node := &LessonNode{NodeBase: NodeBase{Pos: pos}, Title: title}

	// Optional slug as second string
	if p.check(TokString) {
		node.Slug = p.expectString()
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokOrder:
			p.advance()
			node.Order = p.expectInt()
		case TokVideoURL:
			p.advance()
			node.VideoURL = p.expectString()
		case TokContent:
			p.advance()
			node.ContentHTML = p.expectString()
		case TokDraft:
			p.advance()
			if p.check(TokBool) {
				node.IsDraft = p.cur().Literal == "true"
				p.advance()
			} else {
				node.IsDraft = true
			}
		case TokIsDraft:
			p.advance()
			if p.check(TokBool) {
				node.IsDraft = p.cur().Literal == "true"
				p.advance()
			} else {
				node.IsDraft = true
			}
		case TokMediaRef:
			p.advance()
			node.MediaPublicId = p.expectString()
		case TokLMSDuration:
			p.advance()
			node.Duration = p.expectString()
		case TokIsFree:
			p.advance()
			if p.check(TokBool) {
				node.IsFree = p.cur().Literal == "true"
				p.advance()
			} else {
				node.IsFree = true
			}
		case TokDripDays:
			p.advance()
			node.DripDays = p.expectInt()
		case TokDripHours:
			p.advance()
			node.DripHours = p.expectInt()
		case TokDripMinutes:
			p.advance()
			node.DripMinutes = p.expectInt()
		case TokContentGen:
			node.ContentGen = p.parseLMSContentGen()
		case TokDescriptionGen:
			node.ContentGen = p.parseLMSContentGen()
		case TokDurationKw:
			p.advance()
			node.Duration = p.expectString()
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			switch ident {
			case "video_mode":
				if p.check(TokString) {
					node.VideoMode = p.expectString()
				} else if p.isIdentLike() {
					node.VideoMode = p.advanceIdentLike()
				}
			case "video_stub_script":
				node.VideoStubScript = p.expectString()
			case "video_stub_description":
				node.VideoStubDescription = p.expectString()
			case "content_markdown":
				node.ContentMarkdown = p.expectString()
			default:
				p.errorf("unexpected identifier %q in lesson block", ident)
			}
		default:
			p.errorf("unexpected token %s in lesson block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseStringArray is an alias for parseStringList for LMS array fields.
func (p *Parser) parseStringArray() []string {
	return p.parseStringList()
}

// parseLMSQuiz parses: quiz "Title" { pass_threshold 80 max_attempts 3 question "Q?" { ... } }
func (p *Parser) parseLMSQuiz() *LMSQuizNode {
	pos := p.cur().Pos
	p.expect(TokQuiz)
	title := p.expectString()
	node := &LMSQuizNode{NodeBase: NodeBase{Pos: pos}, Title: title}

	// Optional slug as second string
	if p.check(TokString) {
		node.Slug = p.expectString()
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokPassThreshold:
			p.advance()
			node.PassThreshold = p.expectInt()
		case TokMaxAttempts:
			p.advance()
			node.MaxAttempts = p.expectInt()
		case TokQuestion:
			q := p.parseLMSQuestion()
			if q != nil {
				node.Questions = append(node.Questions, q)
			}
		default:
			p.errorf("unexpected token %s in quiz block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseLMSQuestion parses: question "What is Go?" { type multiple_choice options ["A", "B", "C"] answer 0 }
// Named parseLMSQuestion to avoid collision with e-commerce parseQuestion.
func (p *Parser) parseLMSQuestion() *LMSQuestionNode {
	pos := p.cur().Pos
	p.expect(TokQuestion)
	title := p.expectString()
	node := &LMSQuestionNode{NodeBase: NodeBase{Pos: pos}, Title: title, Type: "multiple_choice"}

	// Optional slug as second string
	if p.check(TokString) {
		node.Slug = p.expectString()
	}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokType:
			p.advance()
			if p.check(TokMultipleChoice) {
				node.Type = "multiple_choice"
				p.advance()
			} else if p.check(TokShortAnswer) {
				node.Type = "short_answer"
				p.advance()
			} else if p.check(TokString) {
				node.Type = p.expectString()
			} else if p.isIdentLike() {
				node.Type = p.advanceIdentLike()
			}
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokOptions:
			p.advance()
			node.Options = p.parseStringArray()
		case TokAnswer:
			p.advance()
			if p.check(TokInt) {
				node.Answer = p.expectInt()
			} else if p.check(TokString) {
				node.Answer = p.expectString()
			}
		default:
			p.errorf("unexpected token %s in question block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseLMSContentGen parses: content_gen { instruction "..." references [...] theme "..." }
// Also handles description_gen inside lessons (same structure).
func (p *Parser) parseLMSContentGen() *LMSContentGenNode {
	pos := p.cur().Pos
	p.advance() // consume content_gen or description_gen token
	node := &LMSContentGenNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			switch ident {
			case "instruction":
				node.Instruction = p.expectString()
			case "references":
				node.References = p.parseStringArray()
			case "theme":
				node.Theme = p.expectString()
			default:
				p.errorf("unexpected identifier %q in content_gen block", ident)
			}
		case TokReference:
			p.advance()
			node.References = p.parseStringArray()
		case TokTheme:
			p.advance()
			if p.check(TokString) {
				node.Theme = p.expectString()
			} else if p.isIdentLike() {
				node.Theme = p.advanceIdentLike()
			}
		default:
			p.errorf("unexpected token %s in content_gen block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseDescriptionGen parses: description_gen { instruction "..." references [...] }
func (p *Parser) parseDescriptionGen() *DescriptionGenNode {
	pos := p.cur().Pos
	p.expect(TokDescriptionGen)
	node := &DescriptionGenNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokIdent:
			ident := p.cur().Literal
			p.advance()
			switch ident {
			case "instruction":
				node.Instruction = p.expectString()
			case "references":
				node.References = p.parseStringArray()
			default:
				p.errorf("unexpected identifier %q in description_gen block", ident)
			}
		case TokReference:
			p.advance()
			node.References = p.parseStringArray()
		default:
			p.errorf("unexpected token %s in description_gen block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseCertificate parses: certificate { course_ref "..." template "..." }
func (p *Parser) parseCertificate() *CertificateNode {
	pos := p.cur().Pos
	p.expect(TokCertificate)
	node := &CertificateNode{NodeBase: NodeBase{Pos: pos}}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokCourseRef:
			p.advance()
			node.CourseRef = p.expectString()
		case TokTemplate:
			p.advance()
			node.Template = p.expectString()
		default:
			p.errorf("unexpected token %s in certificate block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}
