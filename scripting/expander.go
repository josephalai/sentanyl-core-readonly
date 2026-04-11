package scripting

import (
	"fmt"
	"strconv"
	"strings"
)

// Expander resolves DSL v2 high-level authoring constructs into the flat
// entity AST that the validator and compiler already understand.
//
// Pipeline position: Parser → **Expander** → Validator → Compiler
//
// The expander performs these transformations in order:
//  1. Collect top-level definitions (default senders, links, patterns, policies)
//  2. Resolve `use sender` statements → propagate from_email/from_name/reply_to
//  3. Resolve `use pattern` statements → expand parameterized enactments/scenes
//  4. Resolve `use policy` statements → expand parameterized triggers
//  5. Expand `scenes 1..3 as var { ... }` → generate concrete SceneNodes
//  6. Resolve symbolic link references in trigger URLs
//  7. Apply inherited defaults to scenes missing sender fields
type Expander struct {
	ast    *ScriptAST
	errors Diagnostics

	// Definition tables populated from top-level blocks
	defaultSenders map[string]*DefaultSenderNode // name → sender (""/"default" for unnamed)
	links          map[string]string             // symbolic name → URL
	patterns       map[string]*PatternDefNode    // name → pattern definition
	policies       map[string]*PolicyDefNode     // name → policy definition
	dataBlocks     map[string]*DataBlockNode     // name → data block (v3)
}

// NewExpander creates an Expander for the given AST.
func NewExpander(ast *ScriptAST) *Expander {
	return &Expander{
		ast:            ast,
		defaultSenders: make(map[string]*DefaultSenderNode),
		links:          make(map[string]string),
		patterns:       make(map[string]*PatternDefNode),
		policies:       make(map[string]*PolicyDefNode),
		dataBlocks:     make(map[string]*DataBlockNode),
	}
}

// Expand runs all expansion passes and returns the modified AST + diagnostics.
// After expansion the AST contains only flat entity nodes (no patterns/policies/use).
func (e *Expander) Expand() (*ScriptAST, Diagnostics) {
	e.collectDefinitions()
	if e.errors.HasErrors() {
		return e.ast, e.errors
	}

	for _, story := range e.ast.Stories {
		e.expandStory(story)
	}

	for _, funnel := range e.ast.Funnels {
		e.expandFunnel(funnel)
	}

	return e.ast, e.errors
}

// ---------- Phase 1: Collect definitions ----------

func (e *Expander) collectDefinitions() {
	// Default senders
	for _, ds := range e.ast.DefaultSenders {
		name := ds.Name
		if name == "" {
			name = "default"
		}
		if _, exists := e.defaultSenders[name]; exists {
			e.errorf(ds.Pos, "duplicate default sender %q", name)
			continue
		}
		e.defaultSenders[name] = ds
	}

	// Links
	if e.ast.Links != nil {
		for name, url := range e.ast.Links.Links {
			e.links[name] = url
		}
	}

	// Patterns
	for _, pat := range e.ast.Patterns {
		if _, exists := e.patterns[pat.Name]; exists {
			e.errorf(pat.Pos, "duplicate pattern definition %q", pat.Name)
			continue
		}
		e.patterns[pat.Name] = pat
	}

	// Policies
	for _, pol := range e.ast.Policies {
		if _, exists := e.policies[pol.Name]; exists {
			e.errorf(pol.Pos, "duplicate policy definition %q", pol.Name)
			continue
		}
		e.policies[pol.Name] = pol
	}

	// Data blocks (v3)
	for _, db := range e.ast.DataBlocks {
		if _, exists := e.dataBlocks[db.Name]; exists {
			e.errorf(db.Pos, "duplicate data block %q", db.Name)
			continue
		}
		e.dataBlocks[db.Name] = db
	}
}

// ---------- Phase 2: Expand story ----------

func (e *Expander) expandStory(story *StoryNode) {
	// Determine effective sender for this story.
	// If no explicit `use sender` is declared, fall back to the unnamed default sender.
	var sender *DefaultSenderNode
	if ds, ok := e.defaultSenders["default"]; ok {
		sender = ds
	}
	for _, us := range story.UseStatements {
		if us.Kind == "sender" {
			target := us.Target
			if target == "" {
				target = "default"
			}
			ds, ok := e.defaultSenders[target]
			if !ok {
				e.errorf(us.Pos, "unknown default sender %q", target)
				continue
			}
			sender = ds
		}
	}

	// Expand for loops at story level → generate storylines
	var allStorylines []*StorylineNode
	allStorylines = append(allStorylines, story.Storylines...)

	for _, fl := range story.ForLoops {
		generated := e.expandForLoopStorylines(fl)
		allStorylines = append(allStorylines, generated...)
	}
	story.Storylines = allStorylines
	story.ForLoops = nil

	// Auto-assign order for storylines if needed
	e.autoAssignStorylineOrders(story.Storylines)

	// Expand each storyline
	for _, sl := range story.Storylines {
		e.expandStoryline(sl, sender)
	}

	// Clear use statements (they've been resolved)
	story.UseStatements = nil
}

// ---------- Phase 3: Expand storyline ----------

func (e *Expander) expandStoryline(sl *StorylineNode, sender *DefaultSenderNode) {
	// Track how many enactments come from direct authoring vs patterns
	directCount := len(sl.Enactments)

	// Process use statements → expand patterns into enactments
	var expandedEnactments []*EnactmentNode

	for _, en := range sl.Enactments {
		expandedEnactments = append(expandedEnactments, en)
	}

	for _, us := range sl.UseStatements {
		switch us.Kind {
		case "pattern":
			enactments := e.expandPattern(us, sender)
			expandedEnactments = append(expandedEnactments, enactments...)
		case "policy":
			// Policies at storyline level don't make sense — they're for enactments
			e.errorf(us.Pos, "'use policy' should be inside an enactment, not a storyline")
		case "sender":
			// Sender at storyline level overrides story-level sender
			target := us.Target
			if target == "" {
				target = "default"
			}
			ds, ok := e.defaultSenders[target]
			if !ok {
				e.errorf(us.Pos, "unknown default sender %q", target)
			} else {
				sender = ds
			}
		}
	}

	// Expand for loops at storyline level → generate enactments
	for _, fl := range sl.ForLoops {
		generated := e.expandForLoopEnactments(fl, sender)
		expandedEnactments = append(expandedEnactments, generated...)
	}
	sl.ForLoops = nil

	sl.Enactments = expandedEnactments
	sl.UseStatements = nil

	// Auto-assign levels only for pattern/loop-generated enactments (not direct authoring)
	if len(expandedEnactments) > directCount {
		e.autoAssignEnactmentLevels(sl.Enactments)
	}

	// Apply enactment_defaults to all enactments
	e.applyEnactmentDefaults(sl.Enactments)

	// Expand each enactment
	for _, en := range sl.Enactments {
		e.expandEnactment(en, sender)
	}
}

// autoAssignEnactmentLevels assigns incrementing level values to enactments
// that don't have explicit levels, or that have duplicate levels from pattern expansion.
func (e *Expander) autoAssignEnactmentLevels(enactments []*EnactmentNode) {
	if len(enactments) == 0 {
		return
	}

	// Check if any enactments have levels set
	usedLevels := make(map[int]bool)
	hasDuplicates := false
	for _, en := range enactments {
		if en.Level != nil {
			if usedLevels[*en.Level] {
				hasDuplicates = true
			}
			usedLevels[*en.Level] = true
		}
	}

	// If there are duplicates or missing levels, auto-assign 1..N
	if hasDuplicates || len(usedLevels) < len(enactments) {
		for i, en := range enactments {
			level := i + 1
			en.Level = &level
		}
	}
}

// autoAssignStorylineOrders assigns incrementing order values to storylines
// that don't have explicit orders.
func (e *Expander) autoAssignStorylineOrders(storylines []*StorylineNode) {
	if len(storylines) == 0 {
		return
	}

	hasUnassigned := false
	for _, sl := range storylines {
		if sl.Order == nil {
			hasUnassigned = true
			break
		}
	}

	if hasUnassigned {
		for i, sl := range storylines {
			if sl.Order == nil {
				order := i + 1
				sl.Order = &order
			}
		}
	}
}

// ---------- Phase 3b: For Loop Expansion (Storylines) ----------

// expandForLoopStorylines expands a for loop inside a story to generate storylines.
func (e *Expander) expandForLoopStorylines(fl *ForNode) []*StorylineNode {
	items := e.resolveForLoopData(fl)
	if items == nil {
		return nil
	}

	var result []*StorylineNode
	for _, item := range items {
		params := e.buildObjectParams(fl.Variable, item)

		for _, slTmpl := range fl.Body {
			sl := &StorylineNode{
				NodeBase: slTmpl.NodeBase,
				Name:     substituteParamsWithDotAccess(slTmpl.Name, params),
			}

			// Copy order if set (literal or expression)
			if slTmpl.Order != nil {
				sl.Order = cloneIntPtr(slTmpl.Order)
			} else if slTmpl.OrderExpr != "" {
				resolved := substituteParamsWithDotAccess(slTmpl.OrderExpr, params)
				v, err := strconv.Atoi(resolved)
				if err != nil {
					e.errorf(slTmpl.Pos, "order expression %q resolved to non-integer %q", slTmpl.OrderExpr, resolved)
				} else {
					sl.Order = &v
				}
			}

			// Clone required badges, lifecycle, etc. with param substitution
			if slTmpl.RequiredBadges != nil {
				sl.RequiredBadges = cloneRequiredBadges(slTmpl.RequiredBadges)
			}
			if slTmpl.OnBegin != nil {
				sl.OnBegin = cloneLifecycleBlockWithDotAccess(slTmpl.OnBegin, params)
			}
			if slTmpl.OnComplete != nil {
				sl.OnComplete = cloneStorylineCompleteBlockWithDotAccess(slTmpl.OnComplete, params)
			}
			if slTmpl.OnFail != nil {
				sl.OnFail = cloneStorylineFailBlockWithDotAccess(slTmpl.OnFail, params)
			}

			// Clone enactments with param substitution
			for _, en := range slTmpl.Enactments {
				sl.Enactments = append(sl.Enactments, e.cloneEnactmentWithDotAccess(en, params))
			}

			// Clone use statements with param substitution
			for _, us := range slTmpl.UseStatements {
				sl.UseStatements = append(sl.UseStatements, e.cloneUseStatementWithDotAccess(us, params))
			}

			// Clone for loops with param substitution
			for _, innerFL := range slTmpl.ForLoops {
				sl.ForLoops = append(sl.ForLoops, e.cloneForLoopWithDotAccess(innerFL, params))
			}

			result = append(result, sl)
		}
	}

	return result
}

// ---------- Phase 3c: For Loop Expansion (Enactments) ----------

// expandForLoopEnactments expands a for loop inside a storyline to generate enactments.
func (e *Expander) expandForLoopEnactments(fl *ForNode, sender *DefaultSenderNode) []*EnactmentNode {
	items := e.resolveForLoopData(fl)
	if items == nil {
		return nil
	}

	var result []*EnactmentNode
	for _, item := range items {
		params := e.buildObjectParams(fl.Variable, item)

		// Expand use statements inside the for body (e.g. use pattern)
		for _, us := range fl.BodyUseStatements {
			expandedUS := e.cloneUseStatementWithDotAccess(us, params)
			switch expandedUS.Kind {
			case "pattern":
				enactments := e.expandPattern(expandedUS, sender)
				result = append(result, enactments...)
			case "policy":
				e.errorf(us.Pos, "'use policy' should be inside an enactment, not a for loop body")
			}
		}

		// Expand enactment templates inside the for body
		for _, enTmpl := range fl.BodyEnactments {
			result = append(result, e.cloneEnactmentWithDotAccess(enTmpl, params))
		}
	}

	return result
}

// ---------- For Loop Data Resolution ----------

// resolveForLoopData resolves the data source for a for loop.
// Returns a list of DataObjectLiterals to iterate over.
func (e *Expander) resolveForLoopData(fl *ForNode) []*DataObjectLiteral {
	// Inline object array
	if len(fl.ObjectItems) > 0 {
		return fl.ObjectItems
	}

	// Inline string list → convert to single-field objects
	if len(fl.Items) > 0 {
		var items []*DataObjectLiteral
		for _, s := range fl.Items {
			items = append(items, &DataObjectLiteral{
				Fields: map[string]string{"_value": s},
			})
		}
		return items
	}

	// Named data block reference
	if fl.DataRef != "" {
		db, ok := e.dataBlocks[fl.DataRef]
		if !ok {
			e.errorf(fl.Pos, "unknown data block %q", fl.DataRef)
			return nil
		}
		return db.Items
	}

	e.errorf(fl.Pos, "for loop has no data source")
	return nil
}

// buildObjectParams builds a parameter map from a data object literal,
// using the loop variable as prefix for dot-access.
// For `for phase in [{ name: "A", link: more_info_a }]`:
//   params = { "phase.name": "A", "phase.link": "https://..." }
func (e *Expander) buildObjectParams(variable string, obj *DataObjectLiteral) map[string]string {
	params := make(map[string]string)
	for k, v := range obj.Fields {
		// Resolve link references in values
		if url, isLink := e.links[v]; isLink {
			params[variable+"."+k] = url
		} else {
			params[variable+"."+k] = v
		}
	}
	return params
}

// ---------- Defaults Application ----------

// applyEnactmentDefaults applies enactment_defaults triggers and policies to all enactments.
func (e *Expander) applyEnactmentDefaults(enactments []*EnactmentNode) {
	if e.ast.EnactmentDefaults == nil && e.ast.SceneDefaults == nil {
		return
	}

	for _, en := range enactments {
		// Apply enactment_defaults
		if e.ast.EnactmentDefaults != nil {
			for _, trTmpl := range e.ast.EnactmentDefaults.Triggers {
				en.Triggers = append(en.Triggers, e.cloneTrigger(trTmpl, nil))
			}
			for _, us := range e.ast.EnactmentDefaults.UseStatements {
				en.UseStatements = append(en.UseStatements, &UseStatementNode{
					NodeBase: us.NodeBase,
					Kind:     us.Kind,
					Target:   us.Target,
					Args:     append([]string{}, us.Args...),
				})
			}
		}

		// Apply scene_defaults (triggers are added at enactment level since
		// scene progression is controlled by enactment triggers)
		if e.ast.SceneDefaults != nil {
			for _, trTmpl := range e.ast.SceneDefaults.Triggers {
				en.Triggers = append(en.Triggers, e.cloneTrigger(trTmpl, nil))
			}
			for _, us := range e.ast.SceneDefaults.UseStatements {
				en.UseStatements = append(en.UseStatements, &UseStatementNode{
					NodeBase: us.NodeBase,
					Kind:     us.Kind,
					Target:   us.Target,
					Args:     append([]string{}, us.Args...),
				})
			}
		}
	}
}

// ---------- Dot-Access Cloning Helpers ----------

// cloneEnactmentWithDotAccess clones an enactment template with dot-access parameter substitution.
func (e *Expander) cloneEnactmentWithDotAccess(tmpl *EnactmentNode, params map[string]string) *EnactmentNode {
	en := &EnactmentNode{
		NodeBase:                    tmpl.NodeBase,
		Name:                        substituteParamsWithDotAccess(tmpl.Name, params),
		Level:                       cloneIntPtr(tmpl.Level),
		Order:                       cloneIntPtr(tmpl.Order),
		SkipToNextStorylineOnExpiry: cloneBoolPtr(tmpl.SkipToNextStorylineOnExpiry),
	}

	// Resolve dot-access level expression
	if tmpl.LevelExpr != "" && en.Level == nil {
		resolved := substituteParamsWithDotAccess(tmpl.LevelExpr, params)
		v, err := strconv.Atoi(resolved)
		if err != nil {
			e.errorf(tmpl.Pos, "level expression %q resolved to non-integer %q", tmpl.LevelExpr, resolved)
		} else {
			en.Level = &v
		}
	}

	// Resolve dot-access order expression
	if tmpl.OrderExpr != "" && en.Order == nil {
		resolved := substituteParamsWithDotAccess(tmpl.OrderExpr, params)
		v, err := strconv.Atoi(resolved)
		if err != nil {
			e.errorf(tmpl.Pos, "order expression %q resolved to non-integer %q", tmpl.OrderExpr, resolved)
		} else {
			en.Order = &v
		}
	}

	for _, sc := range tmpl.Scenes {
		en.Scenes = append(en.Scenes, e.cloneSceneWithDotAccess(sc, params))
	}

	if tmpl.ScenesRange != nil {
		en.ScenesRange = e.cloneScenesRangeWithDotAccess(tmpl.ScenesRange, params)
	}

	for _, tr := range tmpl.Triggers {
		en.Triggers = append(en.Triggers, e.cloneTrigger(tr, params))
	}

	for _, us := range tmpl.UseStatements {
		en.UseStatements = append(en.UseStatements, e.cloneUseStatementWithDotAccess(us, params))
	}

	return en
}

// cloneSceneWithDotAccess clones a scene template with dot-access parameter substitution.
func (e *Expander) cloneSceneWithDotAccess(tmpl *SceneNode, params map[string]string) *SceneNode {
	sc := &SceneNode{
		NodeBase:     tmpl.NodeBase,
		Name:         substituteParamsWithDotAccess(tmpl.Name, params),
		Subject:      substituteParamsWithDotAccess(tmpl.Subject, params),
		Body:         substituteParamsWithDotAccess(tmpl.Body, params),
		FromEmail:    substituteParamsWithDotAccess(tmpl.FromEmail, params),
		FromName:     substituteParamsWithDotAccess(tmpl.FromName, params),
		ReplyTo:      substituteParamsWithDotAccess(tmpl.ReplyTo, params),
		TemplateName: tmpl.TemplateName,
	}
	if tmpl.Tags != nil {
		sc.Tags = append([]string{}, tmpl.Tags...)
	}
	if tmpl.Vars != nil {
		sc.Vars = make(map[string]string)
		for k, v := range tmpl.Vars {
			sc.Vars[k] = substituteParamsWithDotAccess(v, params)
		}
	}
	return sc
}

// cloneScenesRangeWithDotAccess clones a scenes range with dot-access substitution.
func (e *Expander) cloneScenesRangeWithDotAccess(tmpl *ScenesRangeNode, params map[string]string) *ScenesRangeNode {
	sr := &ScenesRangeNode{
		NodeBase:   tmpl.NodeBase,
		RangeStart: tmpl.RangeStart,
		RangeEnd:   tmpl.RangeEnd,
		Variable:   tmpl.Variable,
	}
	if tmpl.Body != nil {
		sr.Body = e.cloneSceneWithDotAccess(tmpl.Body, params)
	}
	return sr
}

// cloneUseStatementWithDotAccess clones a use statement with dot-access substitution.
func (e *Expander) cloneUseStatementWithDotAccess(tmpl *UseStatementNode, params map[string]string) *UseStatementNode {
	us := &UseStatementNode{
		NodeBase: tmpl.NodeBase,
		Kind:     tmpl.Kind,
		Target:   tmpl.Target,
	}
	for _, arg := range tmpl.Args {
		us.Args = append(us.Args, substituteParamsWithDotAccess(arg, params))
	}
	return us
}

// cloneForLoopWithDotAccess clones a for loop with dot-access substitution.
func (e *Expander) cloneForLoopWithDotAccess(tmpl *ForNode, params map[string]string) *ForNode {
	fl := &ForNode{
		NodeBase:   tmpl.NodeBase,
		Variable:   tmpl.Variable,
		DataRef:    tmpl.DataRef,
		RangeStart: tmpl.RangeStart,
		RangeEnd:   tmpl.RangeEnd,
		IsRange:    tmpl.IsRange,
	}

	// Clone items
	for _, item := range tmpl.Items {
		fl.Items = append(fl.Items, substituteParamsWithDotAccess(item, params))
	}
	for _, obj := range tmpl.ObjectItems {
		fl.ObjectItems = append(fl.ObjectItems, obj)
	}

	// Clone body
	for _, sl := range tmpl.Body {
		fl.Body = append(fl.Body, sl) // storyline bodies already resolved
	}
	for _, en := range tmpl.BodyEnactments {
		fl.BodyEnactments = append(fl.BodyEnactments, e.cloneEnactmentWithDotAccess(en, params))
	}
	for _, us := range tmpl.BodyUseStatements {
		fl.BodyUseStatements = append(fl.BodyUseStatements, e.cloneUseStatementWithDotAccess(us, params))
	}

	return fl
}

// ---------- Phase 4: Expand enactment ----------

func (e *Expander) expandEnactment(en *EnactmentNode, sender *DefaultSenderNode) {
	// Expand scenes range if present
	if en.ScenesRange != nil {
		scenes := e.expandScenesRange(en.ScenesRange)
		en.Scenes = append(en.Scenes, scenes...)
		en.ScenesRange = nil
	}

	// Process use statements → expand policies into triggers
	for _, us := range en.UseStatements {
		switch us.Kind {
		case "policy":
			triggers := e.expandPolicy(us)
			en.Triggers = append(en.Triggers, triggers...)
		case "sender":
			target := us.Target
			if target == "" {
				target = "default"
			}
			ds, ok := e.defaultSenders[target]
			if !ok {
				e.errorf(us.Pos, "unknown default sender %q", target)
			} else {
				sender = ds
			}
		case "pattern":
			e.errorf(us.Pos, "'use pattern' should be inside a storyline, not an enactment")
		}
	}
	en.UseStatements = nil

	// Apply sender defaults to scenes
	if sender != nil {
		for _, sc := range en.Scenes {
			e.applySenderDefaults(sc, sender)
		}
	}

	// Resolve symbolic link references in triggers
	e.resolveLinksInTriggers(en.Triggers)

	// Substitute ${link_name} references in scene body and subject text.
	// In the body (HTML context), wrap the URL in an <a> tag so it is
	// clickable and eligible for click-tracking rewriting.
	if len(e.links) > 0 {
		for _, sc := range en.Scenes {
			sc.Subject = substituteParams(sc.Subject, e.links)
			sc.Body = substituteLinksAsHref(sc.Body, e.links)
		}
	}
}

// ---------- Pattern Expansion ----------

func (e *Expander) expandPattern(us *UseStatementNode, sender *DefaultSenderNode) []*EnactmentNode {
	pat, ok := e.patterns[us.Target]
	if !ok {
		e.errorf(us.Pos, "unknown pattern %q", us.Target)
		return nil
	}

	// Build parameter substitution map
	if len(us.Args) != len(pat.Params) {
		e.errorf(us.Pos, "pattern %q expects %d arguments, got %d",
			us.Target, len(pat.Params), len(us.Args))
		return nil
	}

	params := make(map[string]string)
	for i, name := range pat.Params {
		val := us.Args[i]
		// Resolve link references in arguments
		if url, isLink := e.links[val]; isLink {
			params[name] = url
		} else {
			params[name] = val
		}
	}

	var result []*EnactmentNode

	for _, enTmpl := range pat.Enactments {
		en := e.cloneEnactment(enTmpl, params)

		// If the pattern has a scenes range, expand it into this enactment
		if pat.ScenesRange != nil {
			scenes := e.expandScenesRangeWithParams(pat.ScenesRange, params)
			en.Scenes = append(en.Scenes, scenes...)
		}

		// If the pattern has trigger templates, apply them
		for _, trTmpl := range pat.Triggers {
			tr := e.cloneTrigger(trTmpl, params)
			en.Triggers = append(en.Triggers, tr)
		}

		// Apply sender defaults
		if sender != nil {
			for _, sc := range en.Scenes {
				e.applySenderDefaults(sc, sender)
			}
		}

		result = append(result, en)
	}

	return result
}

// ---------- Policy Expansion ----------

func (e *Expander) expandPolicy(us *UseStatementNode) []*TriggerNode {
	pol, ok := e.policies[us.Target]
	if !ok {
		e.errorf(us.Pos, "unknown policy %q", us.Target)
		return nil
	}

	// Build parameter substitution map
	if len(us.Args) != len(pol.Params) {
		e.errorf(us.Pos, "policy %q expects %d arguments, got %d",
			us.Target, len(pol.Params), len(us.Args))
		return nil
	}

	params := make(map[string]string)
	for i, name := range pol.Params {
		val := us.Args[i]
		// Resolve link references in arguments
		if url, isLink := e.links[val]; isLink {
			params[name] = url
		} else {
			params[name] = val
		}
	}

	var result []*TriggerNode
	for _, trTmpl := range pol.Triggers {
		tr := e.cloneTrigger(trTmpl, params)
		result = append(result, tr)
	}
	return result
}

// ---------- Scenes Range Expansion ----------

func (e *Expander) expandScenesRange(sr *ScenesRangeNode) []*SceneNode {
	return e.expandScenesRangeWithParams(sr, nil)
}

func (e *Expander) expandScenesRangeWithParams(sr *ScenesRangeNode, params map[string]string) []*SceneNode {
	if sr == nil || sr.Body == nil {
		return nil
	}

	var scenes []*SceneNode
	for i := sr.RangeStart; i <= sr.RangeEnd; i++ {
		// Build substitution map: merge params + loop variable
		subs := make(map[string]string)
		for k, v := range params {
			subs[k] = v
		}
		subs[sr.Variable] = strconv.Itoa(i)

		sc := e.cloneScene(sr.Body, subs)
		scenes = append(scenes, sc)
	}
	return scenes
}

// ---------- Cloning with substitution ----------

func (e *Expander) cloneEnactment(tmpl *EnactmentNode, params map[string]string) *EnactmentNode {
	en := &EnactmentNode{
		NodeBase:                    tmpl.NodeBase,
		Name:                        substituteParams(tmpl.Name, params),
		Level:                       cloneIntPtr(tmpl.Level),
		Order:                       cloneIntPtr(tmpl.Order),
		SkipToNextStorylineOnExpiry: cloneBoolPtr(tmpl.SkipToNextStorylineOnExpiry),
	}

	for _, sc := range tmpl.Scenes {
		en.Scenes = append(en.Scenes, e.cloneScene(sc, params))
	}

	// Expand scenes range inside the enactment template
	if tmpl.ScenesRange != nil {
		scenes := e.expandScenesRangeWithParams(tmpl.ScenesRange, params)
		en.Scenes = append(en.Scenes, scenes...)
	}

	for _, tr := range tmpl.Triggers {
		en.Triggers = append(en.Triggers, e.cloneTrigger(tr, params))
	}

	// Clone and resolve use statements (e.g. use policy inside pattern enactments)
	for _, us := range tmpl.UseStatements {
		clonedUS := &UseStatementNode{
			NodeBase: us.NodeBase,
			Kind:     us.Kind,
			Target:   us.Target,
		}
		// Substitute parameters in arguments
		for _, arg := range us.Args {
			clonedUS.Args = append(clonedUS.Args, substituteParams(arg, params))
		}
		en.UseStatements = append(en.UseStatements, clonedUS)
	}

	return en
}

func (e *Expander) cloneScene(tmpl *SceneNode, params map[string]string) *SceneNode {
	sc := &SceneNode{
		NodeBase:     tmpl.NodeBase,
		Name:         substituteParams(tmpl.Name, params),
		Subject:      substituteParams(tmpl.Subject, params),
		Body:         substituteParams(tmpl.Body, params),
		FromEmail:    substituteParams(tmpl.FromEmail, params),
		FromName:     substituteParams(tmpl.FromName, params),
		ReplyTo:      substituteParams(tmpl.ReplyTo, params),
		TemplateName: substituteParams(tmpl.TemplateName, params),
	}
	if tmpl.Tags != nil {
		sc.Tags = make([]string, len(tmpl.Tags))
		copy(sc.Tags, tmpl.Tags)
	}
	if tmpl.Vars != nil {
		sc.Vars = make(map[string]string)
		for k, v := range tmpl.Vars {
			sc.Vars[k] = substituteParams(v, params)
		}
	}
	return sc
}

func (e *Expander) cloneTrigger(tmpl *TriggerNode, params map[string]string) *TriggerNode {
	tr := &TriggerNode{
		NodeBase:        tmpl.NodeBase,
		TriggerType:     tmpl.TriggerType,
		UserActionValue: substituteParamsWithDotAccess(tmpl.UserActionValue, params),
		Priority:        cloneIntPtr(tmpl.Priority),
		PersistScope:    tmpl.PersistScope,
		MarkComplete:    tmpl.MarkComplete,
		MarkFailed:      tmpl.MarkFailed,
	}

	if tmpl.Within != nil {
		tr.Within = e.cloneDurationWithParams(tmpl.Within, params)
	}

	if tmpl.RequiredBadges != nil {
		tr.RequiredBadges = cloneRequiredBadges(tmpl.RequiredBadges)
	}

	for _, cond := range tmpl.Conditions {
		tr.Conditions = append(tr.Conditions, cloneCondition(cond))
	}

	for _, act := range tmpl.Actions {
		tr.Actions = append(tr.Actions, e.cloneAction(act, params))
	}

	for _, act := range tmpl.ElseActions {
		tr.ElseActions = append(tr.ElseActions, e.cloneAction(act, params))
	}

	return tr
}

func (e *Expander) cloneAction(tmpl *ActionNode, params map[string]string) *ActionNode {
	act := &ActionNode{
		NodeBase:               tmpl.NodeBase,
		ActionType:             tmpl.ActionType,
		Target:                 substituteParams(tmpl.Target, params),
		SendImmediate:          cloneBoolPtr(tmpl.SendImmediate),
		Unsubscribe:            tmpl.Unsubscribe,
		EndStory:               tmpl.EndStory,
		MarkComplete:           tmpl.MarkComplete,
		MarkFailed:             tmpl.MarkFailed,
		AdvanceToNextStoryline: tmpl.AdvanceToNextStoryline,
		RetryMaxCount:          cloneIntPtr(tmpl.RetryMaxCount),
	}

	if tmpl.Wait != nil {
		act.Wait = e.cloneDurationWithParams(tmpl.Wait, params)
	}

	if tmpl.BadgeTransaction != nil {
		act.BadgeTransaction = cloneBadgeTransactionWithParams(tmpl.BadgeTransaction, params)
	}

	for _, fb := range tmpl.RetryFallback {
		act.RetryFallback = append(act.RetryFallback, e.cloneAction(fb, params))
	}

	return act
}

func (e *Expander) cloneDurationWithParams(tmpl *DurationNode, params map[string]string) *DurationNode {
	d := &DurationNode{
		NodeBase: tmpl.NodeBase,
		Amount:   tmpl.Amount,
		Unit:     tmpl.Unit,
		RawValue: tmpl.RawValue,
	}

	// If the raw value looks like a parameter reference, try to resolve it
	substituted := substituteParams(tmpl.RawValue, params)
	if substituted != tmpl.RawValue {
		// Try to parse the substituted value as a duration
		amount, unit := parseDurationParam(substituted)
		if amount > 0 {
			d.Amount = amount
			d.Unit = unit
			d.RawValue = substituted
		}
	}

	return d
}

// ---------- Link Resolution ----------

func (e *Expander) resolveLinksInTriggers(triggers []*TriggerNode) {
	for _, tr := range triggers {
		// Check if UserActionValue is a symbolic link name
		if tr.UserActionValue != "" {
			if url, ok := e.links[tr.UserActionValue]; ok {
				tr.UserActionValue = url
			}
		}
	}
}

// ---------- Sender Default Application ----------

func (e *Expander) applySenderDefaults(sc *SceneNode, sender *DefaultSenderNode) {
	if sc.FromEmail == "" && sender.FromEmail != "" {
		sc.FromEmail = sender.FromEmail
	}
	if sc.FromName == "" && sender.FromName != "" {
		sc.FromName = sender.FromName
	}
	if sc.ReplyTo == "" && sender.ReplyTo != "" {
		sc.ReplyTo = sender.ReplyTo
	}
}

// ---------- Utility functions ----------

// substituteParams replaces ${param} references in a string.
// If the entire string is exactly a bare parameter name, it's replaced directly.
// This allows policies to use bare parameter names like `on click link`.
func substituteParams(s string, params map[string]string) string {
	if params == nil || s == "" {
		return s
	}

	// Exact match: entire string is a bare parameter name
	if val, ok := params[s]; ok {
		return val
	}

	// Replace ${param} syntax for interpolation
	result := s
	for name, value := range params {
		result = strings.ReplaceAll(result, "${"+name+"}", value)
	}
	return result
}

// substituteLinksAsHref replaces ${link_name} references in an HTML string,
// wrapping the URL in an <a href="..."> tag so it is clickable and eligible
// for click-tracking rewriting.  If the ${link_name} already appears inside
// an href="..." attribute, it is replaced with the raw URL (no double-wrap).
func substituteLinksAsHref(s string, links map[string]string) string {
	if links == nil || s == "" {
		return s
	}
	result := s
	for name, url := range links {
		placeholder := "${" + name + "}"
		// If the placeholder appears inside an existing href="...", just replace with the URL
		if strings.Contains(result, `href="${`+name+`}"`) || strings.Contains(result, `href='${`+name+`}'`) {
			result = strings.ReplaceAll(result, placeholder, url)
		} else {
			// Otherwise wrap in an <a> tag
			result = strings.ReplaceAll(result, placeholder, `<a href="`+url+`">`+url+`</a>`)
		}
	}
	return result
}

// substituteParamsWithDotAccess replaces ${var.field} references in a string.
// Supports both ${var.field} syntax and bare var.field references.
// Falls back to substituteParams for non-dot-access params.
func substituteParamsWithDotAccess(s string, params map[string]string) string {
	if params == nil || s == "" {
		return s
	}

	// Exact match: entire string is a bare parameter name (supports dot access)
	if val, ok := params[s]; ok {
		return val
	}

	// Replace ${var.field} syntax for interpolation
	result := s
	for name, value := range params {
		result = strings.ReplaceAll(result, "${"+name+"}", value)
	}
	return result
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneRequiredBadges(rb *RequiredBadgesNode) *RequiredBadgesNode {
	if rb == nil {
		return nil
	}
	clone := &RequiredBadgesNode{NodeBase: rb.NodeBase}
	if rb.MustHave != nil {
		clone.MustHave = make([]string, len(rb.MustHave))
		copy(clone.MustHave, rb.MustHave)
	}
	if rb.MustNotHave != nil {
		clone.MustNotHave = make([]string, len(rb.MustNotHave))
		copy(clone.MustNotHave, rb.MustNotHave)
	}
	return clone
}

func cloneCondition(cond *ConditionNode) *ConditionNode {
	if cond == nil {
		return nil
	}
	clone := &ConditionNode{
		NodeBase:      cond.NodeBase,
		ConditionType: cond.ConditionType,
		Value:         cond.Value,
	}
	for _, child := range cond.Children {
		clone.Children = append(clone.Children, cloneCondition(child))
	}
	return clone
}

func cloneBadgeTransaction(bt *BadgeTransactionNode) *BadgeTransactionNode {
	return cloneBadgeTransactionWithParams(bt, nil)
}

func cloneBadgeTransactionWithParams(bt *BadgeTransactionNode, params map[string]string) *BadgeTransactionNode {
	if bt == nil {
		return nil
	}
	clone := &BadgeTransactionNode{NodeBase: bt.NodeBase}
	if bt.GiveBadges != nil {
		clone.GiveBadges = make([]string, len(bt.GiveBadges))
		for i, name := range bt.GiveBadges {
			clone.GiveBadges[i] = substituteParams(name, params)
		}
	}
	if bt.RemoveBadges != nil {
		clone.RemoveBadges = make([]string, len(bt.RemoveBadges))
		for i, name := range bt.RemoveBadges {
			clone.RemoveBadges[i] = substituteParams(name, params)
		}
	}
	return clone
}

func cloneBadgeTransactionWithDotAccess(bt *BadgeTransactionNode, params map[string]string) *BadgeTransactionNode {
	if bt == nil {
		return nil
	}
	clone := &BadgeTransactionNode{NodeBase: bt.NodeBase}
	if bt.GiveBadges != nil {
		clone.GiveBadges = make([]string, len(bt.GiveBadges))
		for i, name := range bt.GiveBadges {
			clone.GiveBadges[i] = substituteParamsWithDotAccess(name, params)
		}
	}
	if bt.RemoveBadges != nil {
		clone.RemoveBadges = make([]string, len(bt.RemoveBadges))
		for i, name := range bt.RemoveBadges {
			clone.RemoveBadges[i] = substituteParamsWithDotAccess(name, params)
		}
	}
	return clone
}

func cloneLifecycleBlockWithDotAccess(lb *LifecycleBlock, params map[string]string) *LifecycleBlock {
	if lb == nil {
		return nil
	}
	return &LifecycleBlock{
		NodeBase:         lb.NodeBase,
		BadgeTransaction: cloneBadgeTransactionWithDotAccess(lb.BadgeTransaction, params),
	}
}

func cloneStorylineCompleteBlockWithDotAccess(cb *StorylineCompleteBlock, params map[string]string) *StorylineCompleteBlock {
	if cb == nil {
		return nil
	}
	clone := &StorylineCompleteBlock{
		NodeBase:         cb.NodeBase,
		BadgeTransaction: cloneBadgeTransactionWithDotAccess(cb.BadgeTransaction, params),
		NextStoryline:    substituteParamsWithDotAccess(cb.NextStoryline, params),
	}
	for _, cr := range cb.ConditionalRoutes {
		clone.ConditionalRoutes = append(clone.ConditionalRoutes, cr)
	}
	return clone
}

func cloneStorylineFailBlockWithDotAccess(fb *StorylineFailBlock, params map[string]string) *StorylineFailBlock {
	if fb == nil {
		return nil
	}
	clone := &StorylineFailBlock{
		NodeBase:         fb.NodeBase,
		BadgeTransaction: cloneBadgeTransactionWithDotAccess(fb.BadgeTransaction, params),
		NextStoryline:    substituteParamsWithDotAccess(fb.NextStoryline, params),
	}
	for _, cr := range fb.ConditionalRoutes {
		clone.ConditionalRoutes = append(clone.ConditionalRoutes, cr)
	}
	return clone
}

// parseDurationParam tries to interpret a string like "1d", "2h", "1 day" as a duration.
func parseDurationParam(s string) (int, string) {
	s = strings.TrimSpace(s)

	// Try compact form: 1d, 2h, 30m
	if len(s) >= 2 {
		unit := s[len(s)-1:]
		numStr := s[:len(s)-1]
		if unit == "d" || unit == "h" || unit == "m" || unit == "s" {
			if n, err := strconv.Atoi(strings.TrimSpace(numStr)); err == nil {
				return n, unit
			}
		}
	}

	// Try spaced form: "1 day", "2 hours"
	parts := strings.Fields(s)
	if len(parts) == 2 {
		n, err := strconv.Atoi(parts[0])
		if err == nil {
			switch strings.ToLower(parts[1]) {
			case "day", "days", "d":
				return n, "d"
			case "hour", "hours", "h":
				return n, "h"
			case "minute", "minutes", "m":
				return n, "m"
			case "second", "seconds", "s":
				return n, "s"
			}
		}
	}

	return 0, ""
}

func (e *Expander) errorf(pos Pos, format string, args ...interface{}) {
	e.errors = append(e.errors, Diagnostic{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Level:   DiagError,
	})
}

// ---------- Funnel Expansion ----------

// expandFunnel handles AI context inheritance from funnel → route → stage → page → block.
func (e *Expander) expandFunnel(funnel *FunnelNode) {
	for _, route := range funnel.Routes {
		for _, stage := range route.Stages {
			for _, page := range stage.Pages {
				// Inherit funnel-level AI context if page has none
				if page.AIContext == nil && funnel.AIContext != nil {
					page.AIContext = funnel.AIContext
				}

				for _, block := range page.Blocks {
					// Inherit page-level AI context if block has none
					if block.AIContext == nil && page.AIContext != nil {
						block.AIContext = page.AIContext
					}

					// Propagate AI context URLs to content gen if applicable
					if block.ContentGen != nil && block.AIContext != nil && len(block.ContentGen.ContextURLs) == 0 {
						block.ContentGen.ContextURLs = block.AIContext.ContextURLs
					}
				}
			}
		}
	}
}
