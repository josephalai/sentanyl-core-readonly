package scripting

import (
	"fmt"
	"strings"
)

// SymbolTable tracks named entities and their relationships for resolution.
type SymbolTable struct {
	// Story-level
	StoryNames map[string]Pos

	// Storyline-level: key = "story:storyline"
	StorylineNames map[string]Pos
	StorylineOrder map[string]int // key = "story:storyline"

	// Enactment-level: key = "story:storyline:enactment"
	EnactmentNames     map[string]Pos
	EnactmentLevels    map[string]int
	EnactmentOrders    map[string]int

	// Scene-level: key = "story:storyline:enactment:scene"
	SceneNames map[string]Pos

	// Funnel-level
	FunnelNames map[string]Pos
	RouteNames  map[string]Pos // key = "funnel:route"
	StageNames  map[string]Pos // key = "funnel:route:stage"
	PageNames   map[string]Pos // key = "funnel:route:stage:page"
	FormNames   map[string]Pos // key = "funnel:route:stage:page:form"

	// Site-level
	SiteNames map[string]Pos

	// Badges referenced anywhere
	BadgeNames map[string]bool

	// Tags referenced anywhere
	TagNames map[string]bool

	// Template names referenced
	TemplateNames map[string]bool
}

func newSymbolTable() *SymbolTable {
	return &SymbolTable{
		StoryNames:      make(map[string]Pos),
		StorylineNames:  make(map[string]Pos),
		StorylineOrder:  make(map[string]int),
		EnactmentNames:  make(map[string]Pos),
		EnactmentLevels: make(map[string]int),
		EnactmentOrders: make(map[string]int),
		SceneNames:      make(map[string]Pos),
		FunnelNames:     make(map[string]Pos),
		RouteNames:      make(map[string]Pos),
		StageNames:      make(map[string]Pos),
		PageNames:       make(map[string]Pos),
		FormNames:       make(map[string]Pos),
		SiteNames:       make(map[string]Pos),
		BadgeNames:      make(map[string]bool),
		TagNames:        make(map[string]bool),
		TemplateNames:   make(map[string]bool),
	}
}

// Validator performs semantic analysis on a parsed AST.
type Validator struct {
	ast     *ScriptAST
	symbols *SymbolTable
	errors  Diagnostics
}

// NewValidator creates a Validator.
func NewValidator(ast *ScriptAST) *Validator {
	return &Validator{
		ast:     ast,
		symbols: newSymbolTable(),
	}
}

// Validate runs all validation passes and returns diagnostics.
func (v *Validator) Validate() (*SymbolTable, Diagnostics) {
	// Pass 1: collect all names
	v.collectNames()

	// Pass 2: resolve references
	v.resolveReferences()

	// Pass 3: validate structure
	v.validateStructure()

	return v.symbols, v.errors
}

func (v *Validator) errorf(pos Pos, format string, args ...interface{}) {
	v.errors = append(v.errors, Diagnostic{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Level:   DiagError,
	})
}

func (v *Validator) warnf(pos Pos, format string, args ...interface{}) {
	v.errors = append(v.errors, Diagnostic{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Level:   DiagWarning,
	})
}

// ---------- Pass 1: Collect Names ----------

func (v *Validator) collectNames() {
	for _, story := range v.ast.Stories {
		// Check duplicate story name
		if existing, ok := v.symbols.StoryNames[story.Name]; ok {
			v.errorf(story.Pos, "duplicate story name %q (first defined at %s)", story.Name, existing)
		}
		v.symbols.StoryNames[story.Name] = story.Pos

		// Collect badges from story level
		v.collectBadgesFromRequiredBadges(story.RequiredBadges)
		v.collectBadgesFromLifecycle(story.OnBegin)
		if story.OnComplete != nil && story.OnComplete.BadgeTransaction != nil {
			v.collectBadgesFromBadgeTransaction(story.OnComplete.BadgeTransaction)
		}
		if story.OnFail != nil && story.OnFail.BadgeTransaction != nil {
			v.collectBadgesFromBadgeTransaction(story.OnFail.BadgeTransaction)
		}
		if story.StartTrigger != nil {
			v.symbols.BadgeNames[*story.StartTrigger] = true
		}
		if story.CompleteTrigger != nil {
			v.symbols.BadgeNames[*story.CompleteTrigger] = true
		}

		for _, sl := range story.Storylines {
			slKey := story.Name + ":" + sl.Name

			// Check duplicate storyline name within story
			if existing, ok := v.symbols.StorylineNames[slKey]; ok {
				v.errorf(sl.Pos, "duplicate storyline name %q in story %q (first defined at %s)", sl.Name, story.Name, existing)
			}
			v.symbols.StorylineNames[slKey] = sl.Pos
			if sl.Order != nil {
				v.symbols.StorylineOrder[slKey] = *sl.Order
			}

			v.collectBadgesFromRequiredBadges(sl.RequiredBadges)
			v.collectBadgesFromLifecycle(sl.OnBegin)
			if sl.OnComplete != nil {
				if sl.OnComplete.BadgeTransaction != nil {
					v.collectBadgesFromBadgeTransaction(sl.OnComplete.BadgeTransaction)
				}
				for _, cr := range sl.OnComplete.ConditionalRoutes {
					v.collectBadgesFromRequiredBadges(cr.RequiredBadges)
				}
			}
			if sl.OnFail != nil {
				if sl.OnFail.BadgeTransaction != nil {
					v.collectBadgesFromBadgeTransaction(sl.OnFail.BadgeTransaction)
				}
				for _, cr := range sl.OnFail.ConditionalRoutes {
					v.collectBadgesFromRequiredBadges(cr.RequiredBadges)
				}
			}

			for _, en := range sl.Enactments {
				enKey := slKey + ":" + en.Name

				// Check duplicate enactment name within storyline
				if existing, ok := v.symbols.EnactmentNames[enKey]; ok {
					v.errorf(en.Pos, "duplicate enactment name %q in storyline %q (first defined at %s)", en.Name, sl.Name, existing)
				}
				v.symbols.EnactmentNames[enKey] = en.Pos
				if en.Level != nil {
					v.symbols.EnactmentLevels[enKey] = *en.Level
				}
				if en.Order != nil {
					v.symbols.EnactmentOrders[enKey] = *en.Order
				}

				for _, sc := range en.Scenes {
					scKey := enKey + ":" + sc.Name
					if existing, ok := v.symbols.SceneNames[scKey]; ok {
						v.errorf(sc.Pos, "duplicate scene name %q in enactment %q (first defined at %s)", sc.Name, en.Name, existing)
					}
					v.symbols.SceneNames[scKey] = sc.Pos

					for _, tag := range sc.Tags {
						v.symbols.TagNames[tag] = true
					}
					if sc.TemplateName != "" {
						v.symbols.TemplateNames[sc.TemplateName] = true
					}
				}

				// Collect from triggers
				for _, tr := range en.Triggers {
					v.collectBadgesFromRequiredBadges(tr.RequiredBadges)
					for _, cond := range tr.Conditions {
						v.collectBadgesFromCondition(cond)
					}
					for _, action := range tr.Actions {
						v.collectBadgesFromAction(action)
					}
					for _, action := range tr.ElseActions {
						v.collectBadgesFromAction(action)
					}
				}
			}
		}
	}

	// Collect funnel names
	v.collectFunnelNames()

	// Collect site names
	v.collectSiteNames()
}

func (v *Validator) collectBadgesFromRequiredBadges(rb *RequiredBadgesNode) {
	if rb == nil {
		return
	}
	for _, b := range rb.MustHave {
		v.symbols.BadgeNames[b] = true
	}
	for _, b := range rb.MustNotHave {
		v.symbols.BadgeNames[b] = true
	}
}

func (v *Validator) collectBadgesFromLifecycle(lc *LifecycleBlock) {
	if lc == nil || lc.BadgeTransaction == nil {
		return
	}
	v.collectBadgesFromBadgeTransaction(lc.BadgeTransaction)
}

func (v *Validator) collectBadgesFromBadgeTransaction(bt *BadgeTransactionNode) {
	for _, b := range bt.GiveBadges {
		v.symbols.BadgeNames[b] = true
	}
	for _, b := range bt.RemoveBadges {
		v.symbols.BadgeNames[b] = true
	}
}

func (v *Validator) collectBadgesFromCondition(cond *ConditionNode) {
	if cond == nil {
		return
	}
	if cond.ConditionType == "has_badge" || cond.ConditionType == "not_has_badge" {
		v.symbols.BadgeNames[cond.Value] = true
	}
	if cond.ConditionType == "has_tag" || cond.ConditionType == "not_has_tag" {
		v.symbols.TagNames[cond.Value] = true
	}
	for _, child := range cond.Children {
		v.collectBadgesFromCondition(child)
	}
}

func (v *Validator) collectBadgesFromAction(action *ActionNode) {
	if action == nil {
		return
	}
	if action.BadgeTransaction != nil {
		v.collectBadgesFromBadgeTransaction(action.BadgeTransaction)
	}
	for _, fb := range action.RetryFallback {
		v.collectBadgesFromAction(fb)
	}
}

// ---------- Pass 2: Resolve References ----------

func (v *Validator) resolveReferences() {
	for _, story := range v.ast.Stories {
		// Resolve on_complete/on_fail next_story references
		if story.OnComplete != nil && story.OnComplete.NextStory != "" {
			if _, ok := v.symbols.StoryNames[story.OnComplete.NextStory]; !ok {
				v.errorf(story.OnComplete.Pos, "on_complete references unknown story %q", story.OnComplete.NextStory)
			}
		}
		if story.OnFail != nil && story.OnFail.NextStory != "" {
			if _, ok := v.symbols.StoryNames[story.OnFail.NextStory]; !ok {
				v.errorf(story.OnFail.Pos, "on_fail references unknown story %q", story.OnFail.NextStory)
			}
		}

		for _, sl := range story.Storylines {
			// Resolve storyline on_complete/on_fail next_storyline
			v.resolveStorylineCompleteRef(story, sl)
			v.resolveStorylineFailRef(story, sl)

			for _, en := range sl.Enactments {
				// Resolve trigger action targets
				for _, tr := range en.Triggers {
					v.resolveTriggerActions(story, sl, tr.Actions)
					v.resolveTriggerActions(story, sl, tr.ElseActions)
				}
			}
		}
	}
}

func (v *Validator) resolveStorylineCompleteRef(story *StoryNode, sl *StorylineNode) {
	if sl.OnComplete == nil {
		return
	}
	if sl.OnComplete.NextStoryline != "" {
		key := story.Name + ":" + sl.OnComplete.NextStoryline
		if _, ok := v.symbols.StorylineNames[key]; !ok {
			v.errorf(sl.OnComplete.Pos, "on_complete references unknown storyline %q in story %q", sl.OnComplete.NextStoryline, story.Name)
		}
	}
	for _, cr := range sl.OnComplete.ConditionalRoutes {
		if cr.NextStoryline != "" {
			key := story.Name + ":" + cr.NextStoryline
			if _, ok := v.symbols.StorylineNames[key]; !ok {
				v.errorf(cr.Pos, "conditional_route references unknown storyline %q in story %q", cr.NextStoryline, story.Name)
			}
		}
	}
}

func (v *Validator) resolveStorylineFailRef(story *StoryNode, sl *StorylineNode) {
	if sl.OnFail == nil {
		return
	}
	if sl.OnFail.NextStoryline != "" {
		key := story.Name + ":" + sl.OnFail.NextStoryline
		if _, ok := v.symbols.StorylineNames[key]; !ok {
			v.errorf(sl.OnFail.Pos, "on_fail references unknown storyline %q in story %q", sl.OnFail.NextStoryline, story.Name)
		}
	}
	for _, cr := range sl.OnFail.ConditionalRoutes {
		if cr.NextStoryline != "" {
			key := story.Name + ":" + cr.NextStoryline
			if _, ok := v.symbols.StorylineNames[key]; !ok {
				v.errorf(cr.Pos, "conditional_route references unknown storyline %q in story %q", cr.NextStoryline, story.Name)
			}
		}
	}
}

func (v *Validator) resolveTriggerActions(story *StoryNode, sl *StorylineNode, actions []*ActionNode) {
	for _, action := range actions {
		switch action.ActionType {
		case "jump_to_enactment", "loop_to_enactment", "next_enactment":
			// Resolve within the same storyline
			found := false
			for _, en := range sl.Enactments {
				if en.Name == action.Target {
					found = true
					break
				}
			}
			if !found {
				v.errorf(action.Pos, "action references unknown enactment %q in storyline %q", action.Target, sl.Name)
			}
		case "jump_to_storyline", "loop_to_storyline":
			key := story.Name + ":" + action.Target
			if _, ok := v.symbols.StorylineNames[key]; !ok {
				v.errorf(action.Pos, "action references unknown storyline %q in story %q", action.Target, story.Name)
			}
		}

		// Resolve fallback actions too
		v.resolveTriggerActions(story, sl, action.RetryFallback)
	}
}

// ---------- Pass 3: Validate Structure ----------

func (v *Validator) validateStructure() {
	for _, story := range v.ast.Stories {
		if story.Name == "" {
			v.errorf(story.Pos, "story must have a name")
		}
		if len(story.Storylines) == 0 {
			v.errorf(story.Pos, "story %q must have at least one storyline", story.Name)
		}

		v.validateStoryOrders(story)

		for _, sl := range story.Storylines {
			if sl.Name == "" {
				v.errorf(sl.Pos, "storyline must have a name")
			}
			if len(sl.Enactments) == 0 {
				v.errorf(sl.Pos, "storyline %q must have at least one enactment", sl.Name)
			}

			v.validateEnactmentOrders(story, sl)

			for _, en := range sl.Enactments {
				if en.Name == "" {
					v.errorf(en.Pos, "enactment must have a name")
				}
				if len(en.Scenes) == 0 {
					v.errorf(en.Pos, "enactment %q must have at least one scene", en.Name)
				}

				for _, sc := range en.Scenes {
					v.validateScene(sc, en)
				}

				for _, tr := range en.Triggers {
					v.validateTrigger(tr, en, sl, story)
				}
			}
		}
	}

	// Validate funnel structure
	v.validateFunnelStructure()

	// Validate site structure
	v.validateSiteStructure()
}

func (v *Validator) validateStoryOrders(story *StoryNode) {
	orderSeen := make(map[int]string)
	for _, sl := range story.Storylines {
		if sl.Order != nil {
			if existing, ok := orderSeen[*sl.Order]; ok {
				v.errorf(sl.Pos, "storyline %q has duplicate order %d (conflicts with %q)", sl.Name, *sl.Order, existing)
			}
			orderSeen[*sl.Order] = sl.Name
		}
	}
}

func (v *Validator) validateEnactmentOrders(story *StoryNode, sl *StorylineNode) {
	levelSeen := make(map[int]string)
	orderSeen := make(map[int]string)
	for _, en := range sl.Enactments {
		if en.Level != nil {
			if existing, ok := levelSeen[*en.Level]; ok {
				v.errorf(en.Pos, "enactment %q has duplicate level %d (conflicts with %q) in storyline %q", en.Name, *en.Level, existing, sl.Name)
			}
			levelSeen[*en.Level] = en.Name
		}
		if en.Order != nil {
			if existing, ok := orderSeen[*en.Order]; ok {
				v.errorf(en.Pos, "enactment %q has duplicate order %d (conflicts with %q) in storyline %q", en.Name, *en.Order, existing, sl.Name)
			}
			orderSeen[*en.Order] = en.Name
		}
	}
}

func (v *Validator) validateScene(sc *SceneNode, en *EnactmentNode) {
	if sc.Name == "" {
		v.errorf(sc.Pos, "scene must have a name")
	}
	// Subject is required unless a template is specified
	if sc.Subject == "" && sc.TemplateName == "" {
		v.warnf(sc.Pos, "scene %q has no subject and no template reference", sc.Name)
	}
}

func (v *Validator) validateTrigger(tr *TriggerNode, en *EnactmentNode, sl *StorylineNode, story *StoryNode) {
	// Validate trigger type
	validTypes := map[string]bool{
		"click": true, "not_click": true, "open": true, "not_open": true,
		"sent": true, "webhook": true, "nothing": true, "else": true,
		"bounce": true, "spam": true, "unsubscribe": true, "failure": true,
		"email_validated": true, "user_has_tag": true, "badge": true,
		"submit": true, "abandon": true, "purchase": true,
	}
	if !validTypes[tr.TriggerType] {
		v.errorf(tr.Pos, "unknown trigger type %q", tr.TriggerType)
	}

	// Validate persist scope
	if tr.PersistScope != "" {
		validScopes := map[string]bool{
			"scene": true, "enactment": true, "storyline": true, "story": true, "forever": true,
		}
		if !validScopes[tr.PersistScope] {
			v.errorf(tr.Pos, "invalid persist_scope %q (must be scene, enactment, storyline, story, or forever)", tr.PersistScope)
		}
	}

	// Validate that negative triggers have a within duration
	if strings.HasPrefix(tr.TriggerType, "not_") && tr.Within == nil {
		v.warnf(tr.Pos, "negative trigger %q has no 'within' duration — this may be unintentional", tr.TriggerType)
	}

	// Validate duration units
	if tr.Within != nil {
		v.validateDuration(tr.Within)
	}

	// Validate actions
	for _, action := range tr.Actions {
		v.validateAction(action, en, sl, story)
	}
	for _, action := range tr.ElseActions {
		v.validateAction(action, en, sl, story)
	}

	// Check for mark_complete and mark_failed simultaneously
	if tr.MarkComplete && tr.MarkFailed {
		v.errorf(tr.Pos, "trigger cannot both mark_complete and mark_failed")
	}
}

func (v *Validator) validateAction(action *ActionNode, en *EnactmentNode, sl *StorylineNode, story *StoryNode) {
	validActionTypes := map[string]bool{
		"next_scene": true, "prev_scene": true, "jump_to_enactment": true,
		"jump_to_storyline": true, "advance_to_next_storyline": true,
		"end_story": true, "mark_complete": true, "mark_failed": true,
		"unsubscribe": true, "give_badge": true, "remove_badge": true,
		"retry_scene": true, "retry_enactment": true,
		"loop_to_enactment": true, "loop_to_storyline": true,
		"loop_to_start_enactment": true, "loop_to_start_storyline": true,
		"wait": true, "send_immediate": true, "next_enactment": true,
		"jump_to_stage": true, "start_story": true, "send_email": true,
		"redirect": true, "provide_download": true,
	}
	if !validActionTypes[action.ActionType] {
		v.errorf(action.Pos, "unknown action type %q", action.ActionType)
	}

	// Validate retry has bounds
	if isRetryAction(action.ActionType) {
		if action.RetryMaxCount == nil {
			v.warnf(action.Pos, "retry/loop action %q has no up_to bound — this may create unbounded loops", action.ActionType)
		}
		if action.RetryMaxCount != nil && *action.RetryMaxCount <= 0 {
			v.errorf(action.Pos, "retry/loop action %q has non-positive max count %d", action.ActionType, *action.RetryMaxCount)
		}
	}

	// Validate wait duration
	if action.Wait != nil {
		v.validateDuration(action.Wait)
	}

	// Validate fallback actions
	for _, fb := range action.RetryFallback {
		v.validateAction(fb, en, sl, story)
	}
}

func (v *Validator) validateDuration(d *DurationNode) {
	if d.Amount <= 0 {
		v.errorf(d.Pos, "duration amount must be positive, got %d", d.Amount)
	}
	validUnits := map[string]bool{
		"d": true, "h": true, "m": true, "s": true,
		"days": true, "hours": true, "minutes": true, "seconds": true,
	}
	if !validUnits[d.Unit] {
		v.errorf(d.Pos, "invalid duration unit %q", d.Unit)
	}
}

func isRetryAction(actionType string) bool {
	return actionType == "retry_scene" || actionType == "retry_enactment" ||
		actionType == "loop_to_enactment" || actionType == "loop_to_storyline" ||
		actionType == "loop_to_start_enactment" || actionType == "loop_to_start_storyline"
}

// ---------- Funnel Name Collection ----------

func (v *Validator) collectFunnelNames() {
	for _, funnel := range v.ast.Funnels {
		if existing, ok := v.symbols.FunnelNames[funnel.Name]; ok {
			v.errorf(funnel.Pos, "duplicate funnel name %q (first defined at %s)", funnel.Name, existing)
		}
		v.symbols.FunnelNames[funnel.Name] = funnel.Pos

		for _, route := range funnel.Routes {
			rKey := funnel.Name + ":" + route.Name

			if existing, ok := v.symbols.RouteNames[rKey]; ok {
				v.errorf(route.Pos, "duplicate route name %q in funnel %q (first defined at %s)", route.Name, funnel.Name, existing)
			}
			v.symbols.RouteNames[rKey] = route.Pos

			v.collectBadgesFromRequiredBadges(route.RequiredBadges)

			for _, stage := range route.Stages {
				sKey := rKey + ":" + stage.Name

				if existing, ok := v.symbols.StageNames[sKey]; ok {
					v.errorf(stage.Pos, "duplicate stage name %q in route %q (first defined at %s)", stage.Name, route.Name, existing)
				}
				v.symbols.StageNames[sKey] = stage.Pos

				for _, pg := range stage.Pages {
					pgKey := sKey + ":" + pg.Name

					if existing, ok := v.symbols.PageNames[pgKey]; ok {
						v.errorf(pg.Pos, "duplicate page name %q in stage %q (first defined at %s)", pg.Name, stage.Name, existing)
					}
					v.symbols.PageNames[pgKey] = pg.Pos

					for _, fm := range pg.Forms {
						fmKey := pgKey + ":" + fm.Name
						if existing, ok := v.symbols.FormNames[fmKey]; ok {
							v.errorf(fm.Pos, "duplicate form name %q in page %q (first defined at %s)", fm.Name, pg.Name, existing)
						}
						v.symbols.FormNames[fmKey] = fm.Pos
					}
				}

				for _, tr := range stage.Triggers {
					v.collectBadgesFromRequiredBadges(tr.RequiredBadges)
					for _, cond := range tr.Conditions {
						v.collectBadgesFromCondition(cond)
					}
					for _, action := range tr.Actions {
						v.collectBadgesFromAction(action)
					}
					for _, action := range tr.ElseActions {
						v.collectBadgesFromAction(action)
					}
				}
			}
		}
	}
}

// ---------- Funnel Structure Validation ----------

func (v *Validator) validateFunnelStructure() {
	for _, funnel := range v.ast.Funnels {
		if funnel.Name == "" {
			v.errorf(funnel.Pos, "funnel must have a name")
		}
		if len(funnel.Routes) == 0 {
			v.errorf(funnel.Pos, "funnel %q must have at least one route", funnel.Name)
		}

		for _, route := range funnel.Routes {
			if route.Name == "" {
				v.errorf(route.Pos, "route must have a name")
			}
			if len(route.Stages) == 0 {
				v.errorf(route.Pos, "route %q must have at least one stage", route.Name)
			}

			for _, stage := range route.Stages {
				if stage.Name == "" {
					v.errorf(stage.Pos, "stage must have a name")
				}
				if stage.Path == "" {
					v.warnf(stage.Pos, "stage %q has no path defined", stage.Name)
				}

				for _, pg := range stage.Pages {
					if pg.Name == "" {
						v.errorf(pg.Pos, "page must have a name")
					}
				}

				for _, tr := range stage.Triggers {
					v.validateFunnelTrigger(tr, stage)
				}
			}
		}
	}
}

func (v *Validator) validateFunnelTrigger(tr *TriggerNode, stage *StageNode) {
	validTypes := map[string]bool{
		"submit": true, "abandon": true, "purchase": true,
		"click": true, "webhook": true, "nothing": true, "else": true,
	}
	if !validTypes[tr.TriggerType] {
		v.warnf(tr.Pos, "trigger type %q may not be applicable in funnel context", tr.TriggerType)
	}

	for _, action := range tr.Actions {
		v.validateFunnelAction(action)
	}
	for _, action := range tr.ElseActions {
		v.validateFunnelAction(action)
	}
}

func (v *Validator) validateFunnelAction(action *ActionNode) {
	validActionTypes := map[string]bool{
		"jump_to_stage": true, "start_story": true, "send_email": true,
		"redirect": true, "provide_download": true,
		"give_badge": true, "remove_badge": true, "mark_complete": true,
	}
	if !validActionTypes[action.ActionType] {
		v.warnf(action.Pos, "action type %q may not be applicable in funnel context", action.ActionType)
	}
}

// ---------- Site Validation ----------

func (v *Validator) collectSiteNames() {
	for _, site := range v.ast.Sites {
		if existing, ok := v.symbols.SiteNames[site.Name]; ok {
			v.errorf(site.Pos, "duplicate site name %q (first defined at %s)", site.Name, existing)
		}
		v.symbols.SiteNames[site.Name] = site.Pos

		for _, pg := range site.Pages {
			pgKey := site.Name + ":" + pg.Name
			if existing, ok := v.symbols.PageNames[pgKey]; ok {
				v.errorf(pg.Pos, "duplicate page name %q in site %q (first defined at %s)", pg.Name, site.Name, existing)
			}
			v.symbols.PageNames[pgKey] = pg.Pos
		}
	}
}

func (v *Validator) validateSiteStructure() {
	for _, site := range v.ast.Sites {
		if site.Name == "" {
			v.errorf(site.Pos, "site must have a name")
		}
		if len(site.Pages) == 0 {
			v.warnf(site.Pos, "site %q has no pages defined", site.Name)
		}
		for _, pg := range site.Pages {
			if pg.Name == "" {
				v.errorf(pg.Pos, "page must have a name")
			}
		}
	}
}
