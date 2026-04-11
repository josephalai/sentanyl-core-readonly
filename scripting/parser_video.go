package scripting

// ---------- Video Intelligence Parsers ----------
// Parsers for media, player_preset, channel, media_webhook,
// chapter, turnstile, cta, annotation, and badge_rule declarations.

// parseBool consumes a boolean value from TokBool, TokTrue, or TokFalse.
func (p *Parser) parseBool() bool {
	if p.check(TokBool) {
		val := p.cur().Literal == "true"
		p.advance()
		return val
	}
	if p.check(TokTrue) {
		p.advance()
		return true
	}
	if p.check(TokFalse) {
		p.advance()
		return false
	}
	return true // default
}

// parseMediaDecl parses: media "name" { title "..." source_url "..." ... }
func (p *Parser) parseMediaDecl() *MediaDeclNode {
	pos := p.cur().Pos
	p.expect(TokMedia)
	name := p.expectString()
	node := &MediaDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokDescription:
			p.advance()
			node.Description = p.expectString()
		case TokType:
			p.advance()
			if p.check(TokString) {
				node.Kind = p.expectString()
			} else if p.isIdentLike() {
				node.Kind = p.advanceIdentLike()
			}
		case TokSourceURL:
			p.advance()
			node.SourceURL = p.expectString()
		case TokPosterURL:
			p.advance()
			node.PosterURL = p.expectString()
		case TokPlayerPreset:
			p.advance()
			node.PlayerPreset = p.expectString()
		case TokTags:
			p.advance()
			for p.check(TokString) {
				node.Tags = append(node.Tags, p.expectString())
			}
		case TokChapter:
			ch := p.parseChapterDecl()
			if ch != nil {
				node.Chapters = append(node.Chapters, ch)
			}
		case TokTurnstile:
			inter := p.parseTurnstileDecl()
			if inter != nil {
				node.Interactions = append(node.Interactions, inter)
			}
		case TokCTA:
			inter := p.parseCTADecl()
			if inter != nil {
				node.Interactions = append(node.Interactions, inter)
			}
		case TokAnnotation:
			inter := p.parseAnnotationDecl()
			if inter != nil {
				node.Interactions = append(node.Interactions, inter)
			}
		case TokBadgeRule:
			br := p.parseBadgeRuleDecl()
			if br != nil {
				node.BadgeRules = append(node.BadgeRules, br)
			}
		case TokIdent:
			if p.cur().Literal == "folder" {
				p.advance()
				node.Folder = p.expectString()
			} else {
				p.errorf("unexpected token %s in media block", p.cur().Kind)
				p.advance()
			}
		default:
			p.errorf("unexpected token %s in media block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseChapterDecl parses: chapter "Title" { start_sec 0 end_sec 30 }
func (p *Parser) parseChapterDecl() *ChapterDeclNode {
	pos := p.cur().Pos
	p.expect(TokChapter)
	title := p.expectString()
	node := &ChapterDeclNode{NodeBase: NodeBase{Pos: pos}, Title: title}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokStartSec:
			p.advance()
			node.StartSec = p.expectInt()
		case TokEndSec:
			p.advance()
			node.EndSec = p.expectInt()
		default:
			p.errorf("unexpected token %s in chapter block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseTurnstileDecl parses: turnstile { start_sec 30 required field "email" }
func (p *Parser) parseTurnstileDecl() *InteractionDeclNode {
	pos := p.cur().Pos
	p.expect(TokTurnstile)
	node := &InteractionDeclNode{NodeBase: NodeBase{Pos: pos}, Kind: "turnstile"}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokStartSec:
			p.advance()
			node.StartSec = p.expectInt()
		case TokEndSec:
			p.advance()
			node.EndSec = p.expectInt()
		case TokRequired:
			p.advance()
			node.Required = true
		case TokField:
			p.advance()
			if p.check(TokString) {
				node.Fields = append(node.Fields, p.expectString())
			} else if p.isIdentLike() {
				node.Fields = append(node.Fields, p.advanceIdentLike())
			}
		default:
			p.errorf("unexpected token %s in turnstile block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseCTADecl parses: cta { start_sec 60 text "Click here" url "https://..." }
func (p *Parser) parseCTADecl() *InteractionDeclNode {
	pos := p.cur().Pos
	p.expect(TokCTA)
	node := &InteractionDeclNode{NodeBase: NodeBase{Pos: pos}, Kind: "cta"}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokStartSec:
			p.advance()
			node.StartSec = p.expectInt()
		case TokEndSec:
			p.advance()
			node.EndSec = p.expectInt()
		case TokIdent:
			switch p.cur().Literal {
			case "text":
				p.advance()
				node.Text = p.expectString()
			case "url":
				p.advance()
				node.URL = p.expectString()
			case "button_text":
				p.advance()
				node.ButtonText = p.expectString()
			default:
				p.errorf("unexpected identifier %q in cta block", p.cur().Literal)
				p.advance()
			}
		default:
			p.errorf("unexpected token %s in cta block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseAnnotationDecl parses: annotation { start_sec 45 text "Note" }
func (p *Parser) parseAnnotationDecl() *InteractionDeclNode {
	pos := p.cur().Pos
	p.expect(TokAnnotation)
	node := &InteractionDeclNode{NodeBase: NodeBase{Pos: pos}, Kind: "annotation"}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokStartSec:
			p.advance()
			node.StartSec = p.expectInt()
		case TokEndSec:
			p.advance()
			node.EndSec = p.expectInt()
		case TokIdent:
			switch p.cur().Literal {
			case "text":
				p.advance()
				node.Text = p.expectString()
			case "url":
				p.advance()
				node.URL = p.expectString()
			default:
				p.errorf("unexpected identifier %q in annotation block", p.cur().Literal)
				p.advance()
			}
		default:
			p.errorf("unexpected token %s in annotation block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseBadgeRuleDecl parses: badge_rule { event "progress" operator ">=" threshold 75 badge "engaged" }
func (p *Parser) parseBadgeRuleDecl() *BadgeRuleDeclNode {
	pos := p.cur().Pos
	p.expect(TokBadgeRule)
	node := &BadgeRuleDeclNode{NodeBase: NodeBase{Pos: pos}, Enabled: true}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokIdent:
			switch p.cur().Literal {
			case "event":
				p.advance()
				if p.check(TokString) {
					node.EventName = p.expectString()
				} else if p.isIdentLike() {
					node.EventName = p.advanceIdentLike()
				}
			case "badge":
				p.advance()
				node.BadgeName = p.expectString()
			default:
				p.errorf("unexpected identifier %q in badge_rule block", p.cur().Literal)
				p.advance()
			}
		case TokOperator:
			p.advance()
			if p.check(TokString) {
				node.Operator = p.expectString()
			} else if p.check(TokGT) || p.check(TokLT) || p.check(TokGTEQ) || p.check(TokLTEQ) {
				node.Operator = p.advance().Literal
			}
		case TokThreshold:
			p.advance()
			node.Threshold = p.expectInt()
		case TokBadge:
			p.advance()
			node.BadgeName = p.expectString()
		case TokEnabled:
			p.advance()
			node.Enabled = p.parseBool()
		case TokProgress:
			p.advance()
			node.EventName = "progress"
		case TokComplete:
			p.advance()
			node.EventName = "complete"
		default:
			p.errorf("unexpected token %s in badge_rule block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parsePlayerPresetDecl parses: player_preset "name" { player_color "#3b82f6" ... }
func (p *Parser) parsePlayerPresetDecl() *PlayerPresetDeclNode {
	pos := p.cur().Pos
	p.expect(TokPlayerPreset)
	name := p.expectString()
	node := &PlayerPresetDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokPlayerColor:
			p.advance()
			node.PlayerColor = p.expectString()
		case TokAutoplay:
			p.advance()
			node.Autoplay = p.parseBool()
		case TokIdent:
			switch p.cur().Literal {
			case "show_controls":
				p.advance()
				node.ShowControls = p.parseBool()
			case "show_rewind":
				p.advance()
				node.ShowRewind = p.parseBool()
			case "show_fast_forward":
				p.advance()
				node.ShowFastForward = p.parseBool()
			case "show_skip":
				p.advance()
				node.ShowSkip = p.parseBool()
			case "show_download":
				p.advance()
				node.ShowDownload = p.parseBool()
			case "hide_progress_bar":
				p.advance()
				node.HideProgressBar = p.parseBool()
			case "show_big_play_button":
				p.advance()
				node.ShowBigPlayButton = p.parseBool()
			case "allow_fullscreen":
				p.advance()
				node.AllowFullscreen = p.parseBool()
			case "allow_playback_rate":
				p.advance()
				node.AllowPlaybackRate = p.parseBool()
			case "allow_seeking":
				p.advance()
				node.AllowSeeking = p.parseBool()
			case "muted_default":
				p.advance()
				node.MutedDefault = p.parseBool()
			case "disable_pause":
				p.advance()
				node.DisablePause = p.parseBool()
			case "loop":
				p.advance()
				node.Loop = p.parseBool()
			case "rounded_player":
				p.advance()
				node.RoundedPlayer = p.parseBool()
			case "end_behavior":
				p.advance()
				if p.check(TokString) {
					node.EndBehavior = p.expectString()
				} else if p.isIdentLike() {
					node.EndBehavior = p.advanceIdentLike()
				}
			case "chapter_style":
				p.advance()
				if p.check(TokString) {
					node.ChapterStyle = p.expectString()
				} else if p.isIdentLike() {
					node.ChapterStyle = p.advanceIdentLike()
				}
			case "chapter_position":
				p.advance()
				if p.check(TokString) {
					node.ChapterPosition = p.expectString()
				} else if p.isIdentLike() {
					node.ChapterPosition = p.advanceIdentLike()
				}
			case "chapter_click_jump":
				p.advance()
				node.ChapterClickJump = p.parseBool()
			default:
				p.errorf("unexpected identifier %q in player_preset block", p.cur().Literal)
				p.advance()
			}
		default:
			p.errorf("unexpected token %s in player_preset block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseChannelDecl parses: channel "name" { title "..." items "media1" "media2" }
func (p *Parser) parseChannelDecl() *ChannelDeclNode {
	pos := p.cur().Pos
	p.expect(TokChannel)
	name := p.expectString()
	node := &ChannelDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokTitle:
			p.advance()
			node.Title = p.expectString()
		case TokDescription:
			p.advance()
			node.Description = p.expectString()
		case TokIdent:
			switch p.cur().Literal {
			case "layout":
				p.advance()
				if p.check(TokString) {
					node.Layout = p.expectString()
				} else if p.isIdentLike() {
					node.Layout = p.advanceIdentLike()
				}
			case "items":
				p.advance()
				for p.check(TokString) {
					node.Items = append(node.Items, p.expectString())
				}
			default:
				p.errorf("unexpected identifier %q in channel block", p.cur().Literal)
				p.advance()
			}
		case TokTheme:
			p.advance()
			if p.check(TokString) {
				node.Theme = p.expectString()
			} else if p.isIdentLike() {
				node.Theme = p.advanceIdentLike()
			}
		default:
			p.errorf("unexpected token %s in channel block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}

// parseMediaWebhookDecl parses: media_webhook "name" { url "..." event_types "play" "complete" }
func (p *Parser) parseMediaWebhookDecl() *MediaWebhookDeclNode {
	pos := p.cur().Pos
	p.expect(TokMediaWebhook)
	name := p.expectString()
	node := &MediaWebhookDeclNode{NodeBase: NodeBase{Pos: pos}, Name: name, Enabled: true}

	p.expect(TokLBrace)
	for !p.check(TokRBrace) && !p.atEnd() {
		switch p.cur().Kind {
		case TokIdent:
			switch p.cur().Literal {
			case "url":
				p.advance()
				node.URL = p.expectString()
			case "event_types":
				p.advance()
				for p.check(TokString) {
					node.EventTypes = append(node.EventTypes, p.expectString())
				}
			default:
				p.errorf("unexpected identifier %q in media_webhook block", p.cur().Literal)
				p.advance()
			}
		case TokEnabled:
			p.advance()
			node.Enabled = p.parseBool()
		default:
			p.errorf("unexpected token %s in media_webhook block", p.cur().Kind)
			p.advance()
		}
	}
	p.expect(TokRBrace)
	return node
}
