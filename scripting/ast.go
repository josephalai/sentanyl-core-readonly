package scripting

// ---------- AST Node Types ----------
// Every node embeds NodeBase which carries source position for diagnostics.

// NodeBase provides source position information common to all AST nodes.
type NodeBase struct {
	Pos Pos
}

// ---------- Top-level ----------

// ScriptAST is the root of the parse tree.
type ScriptAST struct {
	NodeBase
	// Top-level definitions (DSL v2)
	DefaultSenders []*DefaultSenderNode
	Links          *LinksBlockNode
	Patterns       []*PatternDefNode
	Policies       []*PolicyDefNode
	// Top-level definitions (DSL v3 generative)
	DataBlocks         []*DataBlockNode
	SceneDefaults      *SceneDefaultsNode
	EnactmentDefaults  *EnactmentDefaultsNode
	// Entity declarations
	Stories        []*StoryNode
	Funnels        []*FunnelNode
	Sites          []*SiteNode
	// E-Commerce declarations
	Products       []*ProductDeclNode
	Offers         []*OfferDeclNode
	Quizzes        []*QuizNode
	// Video Intelligence declarations
	MediaDecls       []*MediaDeclNode
	PlayerPresets    []*PlayerPresetDeclNode
	ChannelDecls     []*ChannelDeclNode
	MediaWebhookDecls []*MediaWebhookDeclNode
}

// ---------- Entity Nodes ----------

// StoryNode represents a `story` declaration.
type StoryNode struct {
	NodeBase
	Name               string
	Priority           *int
	AllowInterruption  *bool
	OnBegin            *LifecycleBlock
	OnComplete         *OnCompleteBlock
	OnFail             *OnFailBlock
	RequiredBadges     *RequiredBadgesNode
	StartTrigger       *string // badge name
	CompleteTrigger    *string // badge name
	Storylines         []*StorylineNode
	ForLoops           []*ForNode          // for loops generating storylines
	DefaultSender      *DefaultSenderNode  // inherited sender for all scenes in story
	UseStatements      []*UseStatementNode  // use sender default, etc.
}

// StorylineNode represents a `storyline` declaration.
type StorylineNode struct {
	NodeBase
	Name              string
	Order             *int
	OrderExpr         string // dot-access expression like "ws.order" (resolved during expansion)
	RequiredBadges    *RequiredBadgesNode
	OnBegin           *LifecycleBlock
	OnComplete        *StorylineCompleteBlock
	OnFail            *StorylineFailBlock
	Enactments        []*EnactmentNode
	ForLoops          []*ForNode          // for loops generating enactments
	UseStatements     []*UseStatementNode // use pattern ..., etc.
}

// EnactmentNode represents an `enactment` declaration.
type EnactmentNode struct {
	NodeBase
	Name                       string
	Level                      *int
	LevelExpr                  string // dot-access expression (resolved during expansion)
	Order                      *int
	OrderExpr                  string // dot-access expression (resolved during expansion)
	SkipToNextStorylineOnExpiry *bool
	Scenes                     []*SceneNode
	Triggers                   []*TriggerNode
	UseStatements              []*UseStatementNode // use policy ..., etc.
	ScenesRange                *ScenesRangeNode    // scenes 1..3 as var { ... }
}

// SceneNode represents a `scene` declaration.
type SceneNode struct {
	NodeBase
	Name         string
	Subject      string
	Body         string
	FromEmail    string
	FromName     string
	ReplyTo      string
	TemplateName string // reference to a template by name
	Vars         map[string]string
	Tags         []string
	Certificate  *CertificateNode // Certificate generation action (LMS)
}

// ---------- Badge / Condition Nodes ----------

// RequiredBadgesNode holds must_have / must_not_have lists.
type RequiredBadgesNode struct {
	NodeBase
	MustHave    []string // badge names
	MustNotHave []string // badge names
}

// BadgeTransactionNode holds give/remove badge operations.
type BadgeTransactionNode struct {
	NodeBase
	GiveBadges   []string // badge names
	RemoveBadges []string // badge names
}

// ---------- Lifecycle Blocks ----------

// LifecycleBlock is used for on_begin (only badge transactions).
type LifecycleBlock struct {
	NodeBase
	BadgeTransaction *BadgeTransactionNode
}

// OnCompleteBlock for Story on_complete.
type OnCompleteBlock struct {
	NodeBase
	BadgeTransaction *BadgeTransactionNode
	NextStory        string // story name reference
}

// OnFailBlock for Story on_fail.
type OnFailBlock struct {
	NodeBase
	BadgeTransaction *BadgeTransactionNode
	NextStory        string // story name reference
}

// StorylineCompleteBlock for Storyline on_complete.
type StorylineCompleteBlock struct {
	NodeBase
	BadgeTransaction  *BadgeTransactionNode
	NextStoryline     string // storyline name reference
	ConditionalRoutes []*ConditionalRouteNode
}

// StorylineFailBlock for Storyline on_fail.
type StorylineFailBlock struct {
	NodeBase
	BadgeTransaction  *BadgeTransactionNode
	NextStoryline     string // storyline name reference
	ConditionalRoutes []*ConditionalRouteNode
}

// ConditionalRouteNode declares a conditional routing rule.
type ConditionalRouteNode struct {
	NodeBase
	RequiredBadges *RequiredBadgesNode
	NextStoryline  string // storyline name reference
	Priority       *int
}

// ---------- Trigger / Action Nodes ----------

// TriggerNode represents a trigger declaration (on click, on not_open, etc.)
type TriggerNode struct {
	NodeBase
	TriggerType     string // "click", "not_click", "open", "not_open", "sent", "webhook", "nothing", "else", "watch", etc.
	UserActionValue string // e.g. link URL for click, tag for user_has_tag
	Within          *DurationNode
	Priority        *int
	PersistScope    string // "scene", "enactment", "storyline", "story", "forever"
	MarkComplete    bool
	MarkFailed      bool
	RequiredBadges  *RequiredBadgesNode
	Conditions      []*ConditionNode
	Actions         []*ActionNode
	ElseActions     []*ActionNode // for else clause
	WatchBlockID    string        // block ID for watch triggers
	WatchOperator   string        // ">", "<", ">=", "<="
	WatchPercent    int           // percentage threshold
}

// ActionNode represents a single action within a trigger.
type ActionNode struct {
	NodeBase
	ActionType       string // "next_scene", "prev_scene", "jump_to_enactment", "jump_to_storyline", etc.
	Target           string // name reference for jumps
	SendImmediate    *bool
	Wait             *DurationNode
	BadgeTransaction *BadgeTransactionNode
	Unsubscribe      bool
	EndStory         bool
	MarkComplete     bool
	MarkFailed       bool
	AdvanceToNextStoryline bool
	RetryMaxCount    *int    // for up_to N
	RetryFallback    []*ActionNode // else clause after retry exhaustion
}

// ConditionNode represents a guard condition on a trigger.
type ConditionNode struct {
	NodeBase
	ConditionType string // "has_badge", "not_has_badge", "has_tag", "not_has_tag", "and", "or", "not"
	Value         string // badge name, tag name, etc.
	Children      []*ConditionNode // for and/or/not
}

// DurationNode represents a time duration like "1d", "2h", "30m".
type DurationNode struct {
	NodeBase
	Amount   int
	Unit     string // "d", "h", "m", "s" or long form "days", "hours", "minutes", "seconds"
	RawValue string // original text e.g. "1d", "2 days"
}

// ---------- Control Flow Nodes (FOR loops for generation) ----------

// ForNode represents a FOR loop for generating multiple entities.
// Supports iterating over data references, inline arrays, and object arrays.
type ForNode struct {
	NodeBase
	Variable       string
	DataRef        string              // reference to a named data block
	Items          []string            // for IN [...] simple string lists
	ObjectItems    []*DataObjectLiteral // for IN [{ key: val }, ...] structured data
	RangeStart     int                 // for RANGE(start, end)
	RangeEnd       int
	IsRange        bool
	Body           []*StorylineNode    // for storyline-level loops (inside story)
	BodyEnactments []*EnactmentNode    // for enactment-level loops (inside storyline)
	BodyScenes     []*SceneNode        // for scene-level loops
	BodyUseStatements []*UseStatementNode // for use pattern/policy inside loop body
}

// DataBlockNode represents a `data name = [...]` top-level definition.
// Provides reusable structured data for iteration.
type DataBlockNode struct {
	NodeBase
	Name    string
	Items   []*DataObjectLiteral
}

// DataObjectLiteral represents `{ key: val, key2: val2, ... }` inside data blocks.
type DataObjectLiteral struct {
	NodeBase
	Fields map[string]string // key → value (string or identifier reference)
}

// SceneDefaultsNode represents `scene_defaults { ... }` with default triggers for all scenes.
type SceneDefaultsNode struct {
	NodeBase
	Triggers      []*TriggerNode
	UseStatements []*UseStatementNode
}

// EnactmentDefaultsNode represents `enactment_defaults { ... }` with default policies for all enactments.
type EnactmentDefaultsNode struct {
	NodeBase
	Triggers      []*TriggerNode
	UseStatements []*UseStatementNode
}

// ========== DSL v2: High-Level Authoring Nodes ==========
// These nodes represent compact authoring constructs that the Expander
// phase resolves into the flat entity AST before validation/compilation.

// DefaultSenderNode represents a `default sender { ... }` block.
// Provides inherited from_email, from_name, reply_to for all scenes.
type DefaultSenderNode struct {
	NodeBase
	Name      string // optional name, e.g. "default" or custom
	FromEmail string
	FromName  string
	ReplyTo   string
}

// LinksBlockNode represents a `links { name = "url" ... }` block.
// Provides named link definitions that can be referenced by symbolic name.
type LinksBlockNode struct {
	NodeBase
	Links map[string]string // name → URL
}

// PatternDefNode represents a `pattern name(params...) { ... }` definition.
// Contains a reusable template body with parameter placeholders.
type PatternDefNode struct {
	NodeBase
	Name       string
	Params     []string          // parameter names
	Enactments []*EnactmentNode  // body: enactment templates
	Scenes     []*SceneNode      // body: scene templates (if pattern generates scenes)
	Triggers   []*TriggerNode    // body: trigger templates
	ScenesRange *ScenesRangeNode // body: scene range generation
}

// PolicyDefNode represents a `policy name(params...) { ... }` definition.
// Contains reusable trigger/action definitions.
type PolicyDefNode struct {
	NodeBase
	Name     string
	Params   []string       // parameter names
	Triggers []*TriggerNode // trigger templates with parameter placeholders
}

// UseStatementNode represents a `use pattern/policy/sender ...` invocation.
type UseStatementNode struct {
	NodeBase
	Kind      string   // "pattern", "policy", "sender"
	Target    string   // name of the definition to use
	Args      []string // arguments passed to the definition (positional)
}

// ScenesRangeNode represents `scenes 1..3 as scene_num { ... }`.
// Generates multiple scenes from a numeric range with a loop variable.
type ScenesRangeNode struct {
	NodeBase
	RangeStart int
	RangeEnd   int
	Variable   string     // loop variable name, e.g. "scene_num"
	Body       *SceneNode // scene template with ${variable} interpolation
}

// StringInterpolation represents a string with ${...} template expressions.
// Used in pattern/scene bodies to substitute parameters.
type StringInterpolation struct {
	NodeBase
	Template string   // original string with ${...} markers
	Vars     []string // extracted variable names in order
}

// ---------- Web Funnel AST Nodes ----------

// FunnelNode represents a `funnel` declaration (parallel to StoryNode for email).
type FunnelNode struct {
	NodeBase
	Name      string
	Domain    string
	AIContext *AIContextNode
	Routes    []*RouteNode
}

// RouteNode represents a `route` declaration inside a funnel (parallel to StorylineNode).
type RouteNode struct {
	NodeBase
	Name           string
	Order          *int
	RequiredBadges *RequiredBadgesNode // reuse existing
	Stages         []*StageNode
}

// LeadMagnetNode represents a `lead_magnet { ... }` declaration inside a stage.
// The compiled output is a pending Asset job that the background hydrator fulfils.
type LeadMagnetNode struct {
	NodeBase
	AssetType   string   // worksheet, guide, cheatsheet, checklist, ebook
	References  []string // source URLs fed to the LLM as context
	Instruction string   // content direction (what to generate)
	Theme       string   // PDF theme: minimal | executive | modern | academic
}

// StageNode represents a `stage` declaration inside a route (parallel to EnactmentNode).
type StageNode struct {
	NodeBase
	Name        string
	Order       *int
	Path        string // e.g. "/free-guide"
	Pages       []*PageNode
	Triggers    []*TriggerNode // reuse existing TriggerNode
	PDFConfig   *PDFConfigNode
	LeadMagnets []*LeadMagnetNode
}

// PageNode represents a `page` declaration inside a stage.
type PageNode struct {
	NodeBase
	Name         string
	TemplateName string
	Blocks       []*BlockNode
	Forms        []*FormNode
	AIContext    *AIContextNode
	LeadMagnets  []*LeadMagnetNode
}

// BlockNode represents a `block` declaration inside a page.
type BlockNode struct {
	NodeBase
	SectionID      string
	BlockType      string // "", "video"
	SourceURL      string // video source URL
	MediaPublicId  string // reference to Media entity
	PlayerPresetId string // reference to PlayerPreset entity
	Autoplay       bool
	ContentGen     *ContentGenNode
	AIContext      *AIContextNode
}

// FormNode represents a `form` declaration inside a page.
type FormNode struct {
	NodeBase
	Name       string
	FormType   string // "lead_capture", "checkout", "upsell", "one_click_upsell"
	Fields     []*FormFieldNode
	ProductID  string
	OfferID    string
	OrderBumps []*OrderBumpNode
}

// FormFieldNode represents a `field` declaration inside a form.
type FormFieldNode struct {
	NodeBase
	FieldName   string
	FieldType   string // email, text, number, card, custom, etc.
	Required    bool
	CustomField string // maps to Contact.CustomFields key (for FieldType "custom")
}

// ContentGenNode represents AI content generation instructions.
type ContentGenNode struct {
	NodeBase
	Length       string   // short, medium, long
	ContextURLs []string
	PromptAppend string
}

// AIContextNode represents `ai context ...` declarations.
type AIContextNode struct {
	NodeBase
	Mode        string   // "global", "extend"
	ContextURLs []string
	ContextRefs []string // named references like "youtube_video_transcript"
}

// PDFConfigNode represents `pdf new ai context ...` declarations.
type PDFConfigNode struct {
	NodeBase
	AIGenerated bool
	AIContext   *AIContextNode
}

// ---------- E-Commerce AST Nodes ----------

// ProductDeclNode represents a `product` declaration (no price - pricing is on Offer).
type ProductDeclNode struct {
	NodeBase
	Name           string
	ProductType    string // course, download, community
	Title          string
	Description    string
	Instructor     string
	ThumbnailURL   string
	Status         string
	DescriptionGen *DescriptionGenNode
	Modules        []*ModuleNode
}

// OfferDeclNode represents an `offer` declaration with pricing and badge grants.
type OfferDeclNode struct {
	NodeBase
	Name             string
	PricingModel     string // free, one_time, payment_plan, recurring
	Price            float64
	Currency         string
	IncludedProducts []string // product names
	GrantedBadges    []string
	OnPurchase       *LifecycleBlock
}

// OrderBumpNode represents an `order_bump` inside a checkout form.
type OrderBumpNode struct {
	NodeBase
	Name    string
	OfferID string
	Text    string
}

// ---------- Quiz AST Nodes ----------

// QuizNode represents a `quiz` declaration.
type QuizNode struct {
	NodeBase
	Name       string
	Questions  []*QuestionNode
	OnComplete *QuizOnCompleteNode
}

// QuestionNode represents a `question` inside a quiz.
type QuestionNode struct {
	NodeBase
	Text    string
	Answers []*AnswerNode
}

// AnswerNode represents an `answer` inside a question.
type AnswerNode struct {
	NodeBase
	Text     string
	AddScore int
}

// QuizOnCompleteNode represents `on complete` logic for a quiz.
type QuizOnCompleteNode struct {
	NodeBase
	ScoreRules []*ScoreRuleNode
}

// ScoreRuleNode represents a score threshold rule in a quiz.
type ScoreRuleNode struct {
	NodeBase
	Operator  string // ">", "<", ">=", "<=", "=="
	Threshold int
	Actions   []*ActionNode // give_badge, etc.
}

// ---------- Website / Site AST Nodes ----------

// SiteNode represents a `site` declaration — a top-level website (parallel to FunnelNode).
type SiteNode struct {
	NodeBase
	Name       string
	Domain     string
	Theme      string
	SEO        *SEONode
	Navigation *NavigationNode
	Pages      []*PageNode
}

// SEONode represents an `seo { ... }` block with meta tags.
type SEONode struct {
	NodeBase
	MetaTitle         string
	MetaDescription   string
	OpenGraphImageURL string
}

// NavigationNode represents a `navigation { header { ... } footer { ... } }` block.
type NavigationNode struct {
	NodeBase
	HeaderLinks map[string]string // label -> path
	FooterLinks map[string]string // label -> path
}

// ---------- LMS (Learning Management System) AST Nodes ----------

// ModuleNode represents a `module` declaration inside a product.
type ModuleNode struct {
	NodeBase
	Slug    string
	Title   string
	Order   int
	Lessons []*LessonNode
	Quizzes []*LMSQuizNode
}

// LessonNode represents a `lesson` declaration inside a module.
type LessonNode struct {
	NodeBase
	Slug          string
	Title         string
	Order         int
	VideoURL      string
	MediaPublicId string
	Duration      string // "HH:MM:SS" format
	ContentHTML   string
	ContentGen    *LMSContentGenNode
	IsFree        bool
	IsDraft       bool
	DripDays      int
}

// LMSQuizNode represents a `quiz` block inside a module (LMS-specific, distinct from e-commerce QuizNode).
// Named LMSQuizNode to avoid collision with e-commerce QuizNode.
type LMSQuizNode struct {
	NodeBase
	Slug          string
	Title         string
	PassThreshold int // Minimum percentage to pass (0-100)
	MaxAttempts   int // 0 = unlimited
	Questions     []*LMSQuestionNode
}

// LMSQuestionNode represents a `question` inside an LMS quiz.
// Named LMSQuestionNode to avoid collision with e-commerce QuestionNode.
type LMSQuestionNode struct {
	NodeBase
	Slug    string
	Type    string      // "multiple_choice" | "short_answer"
	Title   string      // Question text
	Options []string    // Answer options (for multiple_choice)
	Answer  interface{} // int (option index) for multiple_choice, string for short_answer
}

// LMSContentGenNode represents a `content_gen` block inside a lesson (LMS-specific).
// Distinct from ContentGenNode which has Length/ContextURLs/PromptAppend.
type LMSContentGenNode struct {
	NodeBase
	Instruction string
	References  []string
	Theme       string
}

// DescriptionGenNode represents a `description_gen` block inside a course product.
type DescriptionGenNode struct {
	NodeBase
	Instruction string
	References  []string
}

// CertificateNode represents a `certificate` block inside a story scene.
type CertificateNode struct {
	NodeBase
	CourseRef string // Product slug reference
	Template  string // Certificate template identifier
}

// ---------- Video Intelligence AST Nodes ----------

// MediaDeclNode represents a `media` declaration — a first-class video/audio asset.
type MediaDeclNode struct {
	NodeBase
	Name          string
	Title         string
	Description   string
	Kind          string // "video" | "audio"
	SourceURL     string
	PosterURL     string
	Chapters      []*ChapterDeclNode
	Interactions  []*InteractionDeclNode
	BadgeRules    []*BadgeRuleDeclNode
	PlayerPreset  string // reference by name
	Tags          []string
	Folder        string
}

// ChapterDeclNode represents a `chapter` inside a media declaration.
type ChapterDeclNode struct {
	NodeBase
	Title    string
	StartSec int
	EndSec   int
}

// InteractionDeclNode represents an interaction (turnstile, cta, annotation) inside media.
type InteractionDeclNode struct {
	NodeBase
	Kind     string // "turnstile", "cta", "annotation"
	StartSec int
	EndSec   int
	// Turnstile fields
	Required bool
	Fields   []string
	// CTA/Annotation fields
	Text       string
	URL        string
	ButtonText string
}

// BadgeRuleDeclNode represents a `badge_rule` inside a media declaration.
type BadgeRuleDeclNode struct {
	NodeBase
	EventName string // "progress", "complete", "cta_click", etc.
	Operator  string // ">", ">=", "<", "<=", "=="
	Threshold int
	BadgeName string
	Enabled   bool
}

// PlayerPresetDeclNode represents a `player_preset` declaration.
type PlayerPresetDeclNode struct {
	NodeBase
	Name              string
	// Appearance
	PlayerColor       string
	ShowControls      bool

	// Show Buttons settings
	ShowRewind        bool
	ShowFastForward   bool
	ShowSkip          bool
	ShowDownload      bool
	HideProgressBar   bool

	// Other controls
	ShowBigPlayButton bool
	AllowFullscreen   bool
	AllowPlaybackRate bool
	AllowSeeking      bool

	// Behaviour
	Autoplay          bool
	MutedDefault      bool
	DisablePause      bool
	Loop              bool

	// End behaviour
	EndBehavior string // "stop" | "loop" | "prevent_replay"

	// Player style
	RoundedPlayer bool

	// Chapter controls
	ChapterStyle      string // "hover" | "buttons"
	ChapterPosition   string
	ChapterClickJump  bool
}

// ChannelDeclNode represents a `channel` declaration.
type ChannelDeclNode struct {
	NodeBase
	Name        string
	Title       string
	Description string
	Layout      string
	Theme       string
	Items       []string // media names
}

// MediaWebhookDeclNode represents a `media_webhook` declaration.
type MediaWebhookDeclNode struct {
	NodeBase
	Name       string
	URL        string
	EventTypes []string
	Enabled    bool
}
