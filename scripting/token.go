package scripting

import "fmt"

// Pos records where in the source a token appeared.
type Pos struct {
	Line   int // 1-based
	Col    int // 1-based (byte offset in line)
	Offset int // 0-based byte offset in source
}

func (p Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

// TokenKind enumerates every token the lexer can emit.
type TokenKind int

const (
	// Special
	TokEOF TokenKind = iota
	TokIllegal

	// Literals
	TokIdent   // bare identifier
	TokString  // "..." quoted string
	TokInt     // integer literal
	TokFloat   // float literal (unused for now but available)
	TokBool    // true | false
	TokDuration // e.g. 1d, 2h, 30m, 1h30m

	// Punctuation / operators
	TokLBrace    // {
	TokRBrace    // }
	TokLParen    // (
	TokRParen    // )
	TokLBracket  // [
	TokRBracket  // ]
	TokComma     // ,
	TokDot       // .
	TokColon     // :
	TokEquals    // =
	TokBang      // !
	TokLT        // <
	TokGT        // >
	TokLTEQ      // <=
	TokGTEQ      // >=
	TokEQEQ      // ==
	TokNEQ       // !=
	TokAmpAmp    // &&
	TokPipePipe  // ||
	TokPlus      // +
	TokMinus     // -
	TokStar      // *
	TokSlash     // /

	// Keywords — entity level
	TokStory
	TokStoryline
	TokEnactment
	TokScene
	TokMessage
	TokTemplate
	TokEmail

	// Keywords — structural
	TokName
	TokPriority
	TokOrder
	TokLevel
	TokAllowInterruption
	TokSkipToNextStorylineOnExpiry
	TokSubject
	TokBody
	TokFromEmail
	TokFromName
	TokReplyTo
	TokTags
	TokVars

	// Keywords — lifecycle
	TokOnBegin
	TokOnComplete
	TokOnFail

	// Keywords — badges
	TokRequiredBadges
	TokMustHave
	TokMustNotHave
	TokGiveBadge
	TokRemoveBadge
	TokBadge
	TokStartTrigger
	TokCompleteTrigger

	// Keywords — transitions
	TokNextStory
	TokNextStoryline
	TokNextEnactment
	TokAdvanceToNextStoryline
	TokEndStory
	TokMarkComplete
	TokMarkFailed

	// Keywords — triggers / actions
	TokOn
	TokDo
	TokWhen
	TokWithin
	TokClick
	TokNotClick
	TokOpen
	TokNotOpen
	TokSent
	TokWebhook
	TokNothing
	TokElse
	TokBounce
	TokSpam
	TokUnsubscribe
	TokFailure
	TokEmailValidated
	TokUserHasTag

	// Keywords — action modifiers
	TokSendImmediate
	TokWait
	TokJumpToEnactment
	TokJumpToStoryline
	TokNextScene
	TokPrevScene
	TokRetryScene
	TokRetryEnactment
	TokLoopToEnactment
	TokLoopToStoryline
	TokLoopToStartEnactment
	TokLoopToStartStoryline
	TokUpTo
	TokTimes

	// Keywords — trigger config
	TokPersistScope
	TokTriggerPriority

	// Keywords — conditional routing
	TokConditionalRoute
	TokRoute

	// Keywords — conditions
	TokHasBadge
	TokNotHasBadge
	TokHasTag
	TokNotHasTag
	TokIf
	TokElseIf
	TokAnd
	TokOr
	TokNot

	// Keywords — control flow
	TokFor
	TokIn
	TokRange
	TokWhile
	TokBreak
	TokContinue

	// Keywords — DSL v2 (defaults, patterns, policies, links)
	TokDefault
	TokSender
	TokLinks
	TokPattern
	TokPolicy
	TokUse
	TokAs
	TokScenes   // `scenes` keyword for scene ranges
	TokDotDot   // `..` range operator
	TokDollarLBrace // `${` for string interpolation start

	// Keywords — DSL v3 (generative: data blocks, defaults blocks)
	TokData            // `data` keyword for top-level data definitions
	TokSceneDefaults   // `scene_defaults` keyword
	TokEnactmentDefaults // `enactment_defaults` keyword

	// Keywords — web funnel entities
	TokFunnel
	TokStage
	TokPage
	TokBlock
	TokForm
	TokField
	TokPath
	TokDomain
	TokAI
	TokContext
	TokGlobal
	TokExtend
	TokLength
	TokPrompt
	TokSubmit
	TokAbandon
	TokPurchase
	TokProductId
	TokType
	TokCheckout
	TokUpsell
	TokLeadCapture
	TokRequired
	TokNew
	TokPdf
	TokLeadMagnet // lead_magnet
	TokReference  // reference
	TokJumpToStage
	TokStartStory
	TokSendEmailAction
	TokRedirect
	TokProvideDownload
	TokMustHaveBadge
	TokMustNotHaveBadge

	// Keywords — video / watch tracking
	TokWatch
	TokVideo
	TokAutoplay
	TokSourceURL
	TokPercent // %

	// Keywords — e-commerce & advanced funnel
	TokProduct     // product (top-level declaration)
	TokOffer       // offer (top-level declaration)
	TokPricingModel
	TokPrice
	TokCurrency
	TokIncludesProduct
	TokGrantsBadge
	TokOrderBump
	TokOneClickUpsell
	TokDecline
	TokCustom    // custom field type
	TokQuiz
	TokQuestion
	TokAnswer
	TokAddScore
	TokScore

	// Keywords — site / website
	TokSite        // site (top-level website declaration)
	TokSEO         // seo
	TokNavigation  // navigation
	TokHeader      // header
	TokFooter      // footer
	TokTheme       // theme
	TokTitle       // title (inside seo block)
	TokDescription // description (inside seo and product blocks)

	// Keywords — LMS (Learning Management System)
	TokModule    // module
	TokLesson    // lesson
	TokVideoURL  // video_url
	TokContent   // content
	TokDraft     // draft
	TokInstructor    // instructor
	TokLMSDuration   // duration
	TokIsFree        // is_free
	TokIsDraft       // is_draft
	TokDripDays      // drip_days
	TokContentGen    // content_gen
	TokDescriptionGen // description_gen
	TokPassThreshold // pass_threshold
	TokMaxAttempts   // max_attempts
	TokOptions       // options
	TokMultipleChoice // multiple_choice
	TokShortAnswer   // short_answer
	TokCertificate   // certificate
	TokCourseRef     // course_ref
	TokCourse        // course (product type identifier)
	TokDurationKw    // duration (LMS lesson duration, alias for TokLMSDuration)

	// Keywords — Video Intelligence
	TokMedia         // media
	TokPlayerPreset  // player_preset
	TokChannel       // channel
	TokChapter       // chapter
	TokTurnstile     // turnstile
	TokCTA           // cta
	TokAnnotation    // annotation
	TokBadgeRule     // badge_rule
	TokMediaWebhook  // media_webhook
	TokMediaRef      // media_ref
	TokStartSec      // start_sec
	TokEndSec        // end_sec
	TokThreshold     // threshold
	TokOperator      // operator
	TokEnabled       // enabled
	TokProgress      // progress
	TokComplete      // complete
	TokPlay          // play
	TokPause         // pause
	TokRewatch       // rewatch
	TokPosterURL     // poster_url
	TokPlayerColor   // player_color

	// Keywords — misc
	TokTrue
	TokFalse
	TokNil
)

var tokenNames = map[TokenKind]string{
	TokEOF:     "EOF",
	TokIllegal: "ILLEGAL",
	TokIdent:   "IDENT",
	TokString:  "STRING",
	TokInt:     "INT",
	TokFloat:   "FLOAT",
	TokBool:    "BOOL",
	TokDuration: "DURATION",

	TokLBrace:   "{",
	TokRBrace:   "}",
	TokLParen:   "(",
	TokRParen:   ")",
	TokLBracket: "[",
	TokRBracket: "]",
	TokComma:    ",",
	TokDot:      ".",
	TokColon:    ":",
	TokEquals:   "=",
	TokBang:     "!",
	TokLT:       "<",
	TokGT:       ">",
	TokLTEQ:     "<=",
	TokGTEQ:     ">=",
	TokEQEQ:     "==",
	TokNEQ:      "!=",
	TokAmpAmp:   "&&",
	TokPipePipe: "||",
	TokPlus:     "+",
	TokMinus:    "-",
	TokStar:     "*",
	TokSlash:    "/",

	TokStory:      "story",
	TokStoryline:  "storyline",
	TokEnactment:  "enactment",
	TokScene:      "scene",
	TokMessage:    "message",
	TokTemplate:   "template",
	TokEmail:      "email",

	TokName:                        "name",
	TokPriority:                    "priority",
	TokOrder:                       "order",
	TokLevel:                       "level",
	TokAllowInterruption:           "allow_interruption",
	TokSkipToNextStorylineOnExpiry: "skip_to_next_storyline_on_expiry",
	TokSubject:                     "subject",
	TokBody:                        "body",
	TokFromEmail:                   "from_email",
	TokFromName:                    "from_name",
	TokReplyTo:                     "reply_to",
	TokTags:                        "tags",
	TokVars:                        "vars",

	TokOnBegin:    "on_begin",
	TokOnComplete: "on_complete",
	TokOnFail:     "on_fail",

	TokRequiredBadges:  "required_badges",
	TokMustHave:        "must_have",
	TokMustNotHave:     "must_not_have",
	TokGiveBadge:       "give_badge",
	TokRemoveBadge:     "remove_badge",
	TokBadge:           "badge",
	TokStartTrigger:    "start_trigger",
	TokCompleteTrigger: "complete_trigger",

	TokNextStory:               "next_story",
	TokNextStoryline:           "next_storyline",
	TokNextEnactment:           "next_enactment",
	TokAdvanceToNextStoryline:  "advance_to_next_storyline",
	TokEndStory:                "end_story",
	TokMarkComplete:            "mark_complete",
	TokMarkFailed:              "mark_failed",

	TokOn:     "on",
	TokDo:     "do",
	TokWhen:   "when",
	TokWithin: "within",

	TokClick:          "click",
	TokNotClick:       "not_click",
	TokOpen:           "open",
	TokNotOpen:        "not_open",
	TokSent:           "sent",
	TokWebhook:        "webhook",
	TokNothing:        "nothing",
	TokElse:           "else",
	TokBounce:         "bounce",
	TokSpam:           "spam",
	TokUnsubscribe:    "unsubscribe",
	TokFailure:        "failure",
	TokEmailValidated: "email_validated",
	TokUserHasTag:     "user_has_tag",

	TokSendImmediate:        "send_immediate",
	TokWait:                 "wait",
	TokJumpToEnactment:      "jump_to_enactment",
	TokJumpToStoryline:      "jump_to_storyline",
	TokNextScene:            "next_scene",
	TokPrevScene:            "prev_scene",
	TokRetryScene:           "retry_scene",
	TokRetryEnactment:       "retry_enactment",
	TokLoopToEnactment:      "loop_to_enactment",
	TokLoopToStoryline:      "loop_to_storyline",
	TokLoopToStartEnactment: "loop_to_start_enactment",
	TokLoopToStartStoryline: "loop_to_start_storyline",
	TokUpTo:                 "up_to",
	TokTimes:                "times",

	TokPersistScope:    "persist_scope",
	TokTriggerPriority: "trigger_priority",

	TokConditionalRoute: "conditional_route",
	TokRoute:            "route",

	TokHasBadge:    "has_badge",
	TokNotHasBadge: "not_has_badge",
	TokHasTag:      "has_tag",
	TokNotHasTag:   "not_has_tag",
	TokIf:          "if",
	TokElseIf:      "else_if",
	TokAnd:         "and",
	TokOr:          "or",
	TokNot:         "not",

	TokFor:      "for",
	TokIn:       "in",
	TokRange:    "range",
	TokWhile:    "while",
	TokBreak:    "break",
	TokContinue: "continue",

	TokDefault:       "default",
	TokSender:        "sender",
	TokLinks:         "links",
	TokPattern:       "pattern",
	TokPolicy:        "policy",
	TokUse:           "use",
	TokAs:            "as",
	TokScenes:        "scenes",
	TokDotDot:        "..",
	TokDollarLBrace:  "${",

	TokData:              "data",
	TokSceneDefaults:     "scene_defaults",
	TokEnactmentDefaults: "enactment_defaults",

	TokFunnel:          "funnel",
	TokStage:           "stage",
	TokPage:            "page",
	TokBlock:           "block",
	TokForm:            "form",
	TokField:           "field",
	TokPath:            "path",
	TokDomain:          "domain",
	TokAI:              "ai",
	TokContext:         "context",
	TokGlobal:          "global",
	TokExtend:          "extend",
	TokLength:          "length",
	TokPrompt:          "prompt",
	TokSubmit:          "submit",
	TokAbandon:         "abandon",
	TokPurchase:        "purchase",
	TokProductId:       "product_id",
	TokType:            "type",
	TokCheckout:        "checkout",
	TokUpsell:          "upsell",
	TokLeadCapture:     "lead_capture",
	TokRequired:        "required",
	TokNew:             "new",
	TokPdf:             "pdf",
	TokLeadMagnet:      "lead_magnet",
	TokReference:       "reference",
	TokJumpToStage:     "jump_to_stage",
	TokStartStory:      "start_story",
	TokSendEmailAction: "send_email",
	TokRedirect:        "redirect",
	TokProvideDownload: "provide_download",
	TokMustHaveBadge:   "must_have_badge",
	TokMustNotHaveBadge: "must_not_have_badge",

	TokWatch:     "watch",
	TokVideo:     "video",
	TokAutoplay:  "autoplay",
	TokSourceURL: "source_url",
	TokPercent:   "%",

	TokTrue:  "true",
	TokFalse: "false",
	TokNil:   "nil",

	// Site / website tokens
	TokSite:        "site",
	TokSEO:         "seo",
	TokNavigation:  "navigation",
	TokHeader:      "header",
	TokFooter:      "footer",
	TokTheme:       "theme",
	TokTitle:       "title",
	TokDescription: "description",

	// LMS tokens
	TokModule:   "module",
	TokLesson:   "lesson",
	TokVideoURL: "video_url",
	TokContent:  "content",
	TokDraft:    "draft",
	TokInstructor:    "instructor",
	TokLMSDuration:   "duration",
	TokIsFree:        "is_free",
	TokIsDraft:       "is_draft",
	TokDripDays:      "drip_days",
	TokContentGen:    "content_gen",
	TokDescriptionGen: "description_gen",
	TokPassThreshold: "pass_threshold",
	TokMaxAttempts:   "max_attempts",
	TokOptions:       "options",
	TokMultipleChoice: "multiple_choice",
	TokShortAnswer:   "short_answer",
	TokCertificate:   "certificate",
	TokCourseRef:     "course_ref",
	TokCourse:        "course",
	TokDurationKw:    "duration",

	// Video Intelligence tokens
	TokMedia:        "media",
	TokPlayerPreset: "player_preset",
	TokChannel:      "channel",
	TokChapter:      "chapter",
	TokTurnstile:    "turnstile",
	TokCTA:          "cta",
	TokAnnotation:   "annotation",
	TokBadgeRule:    "badge_rule",
	TokMediaWebhook: "media_webhook",
	TokMediaRef:     "media_ref",
	TokStartSec:     "start_sec",
	TokEndSec:       "end_sec",
	TokThreshold:    "threshold",
	TokOperator:     "operator",
	TokEnabled:      "enabled",
	TokProgress:     "progress",
	TokComplete:     "complete",
	TokPlay:         "play",
	TokPause:        "pause",
	TokRewatch:      "rewatch",
	TokPosterURL:    "poster_url",
	TokPlayerColor:  "player_color",

	// E-commerce & advanced funnel tokens
	TokProduct:         "product",
	TokOffer:           "offer",
	TokPricingModel:    "pricing_model",
	TokPrice:           "price",
	TokCurrency:        "currency",
	TokIncludesProduct: "includes_product",
	TokGrantsBadge:     "grants_badge",
	TokOrderBump:       "order_bump",
	TokOneClickUpsell:  "one_click_upsell",
	TokDecline:         "decline",
	TokCustom:          "custom",
	TokQuiz:            "quiz",
	TokQuestion:        "question",
	TokAnswer:          "answer",
	TokAddScore:        "add_score",
	TokScore:           "score",
}

func (k TokenKind) String() string {
	if s, ok := tokenNames[k]; ok {
		return s
	}
	return fmt.Sprintf("TokenKind(%d)", int(k))
}

// Token is a single lexical unit.
type Token struct {
	Kind    TokenKind
	Literal string
	Pos     Pos
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %s)", t.Kind, t.Literal, t.Pos)
}

// keywords maps lowercase identifiers to their keyword token kind.
var keywords = map[string]TokenKind{
	"story":                          TokStory,
	"storyline":                      TokStoryline,
	"enactment":                      TokEnactment,
	"scene":                          TokScene,
	"message":                        TokMessage,
	"template":                       TokTemplate,
	"email":                          TokEmail,
	"name":                           TokName,
	"priority":                       TokPriority,
	"order":                          TokOrder,
	"level":                          TokLevel,
	"allow_interruption":             TokAllowInterruption,
	"skip_to_next_storyline_on_expiry": TokSkipToNextStorylineOnExpiry,
	"subject":                        TokSubject,
	"body":                           TokBody,
	"from_email":                     TokFromEmail,
	"from_name":                      TokFromName,
	"reply_to":                       TokReplyTo,
	"tags":                           TokTags,
	"vars":                           TokVars,
	"on_begin":                       TokOnBegin,
	"on_complete":                    TokOnComplete,
	"on_fail":                        TokOnFail,
	"required_badges":                TokRequiredBadges,
	"must_have":                      TokMustHave,
	"must_not_have":                  TokMustNotHave,
	"give_badge":                     TokGiveBadge,
	"remove_badge":                   TokRemoveBadge,
	"badge":                          TokBadge,
	"start_trigger":                  TokStartTrigger,
	"complete_trigger":               TokCompleteTrigger,
	"next_story":                     TokNextStory,
	"next_storyline":                 TokNextStoryline,
	"next_enactment":                 TokNextEnactment,
	"advance_to_next_storyline":      TokAdvanceToNextStoryline,
	"end_story":                      TokEndStory,
	"mark_complete":                  TokMarkComplete,
	"mark_failed":                    TokMarkFailed,
	"on":                             TokOn,
	"do":                             TokDo,
	"when":                           TokWhen,
	"within":                         TokWithin,
	"click":                          TokClick,
	"not_click":                      TokNotClick,
	"open":                           TokOpen,
	"not_open":                       TokNotOpen,
	"sent":                           TokSent,
	"webhook":                        TokWebhook,
	"nothing":                        TokNothing,
	"else":                           TokElse,
	"bounce":                         TokBounce,
	"spam":                           TokSpam,
	"unsubscribe":                    TokUnsubscribe,
	"failure":                        TokFailure,
	"email_validated":                TokEmailValidated,
	"user_has_tag":                   TokUserHasTag,
	"send_immediate":                 TokSendImmediate,
	"wait":                           TokWait,
	"jump_to_enactment":              TokJumpToEnactment,
	"jump_to_storyline":              TokJumpToStoryline,
	"next_scene":                     TokNextScene,
	"prev_scene":                     TokPrevScene,
	"retry_scene":                    TokRetryScene,
	"retry_enactment":                TokRetryEnactment,
	"loop_to_enactment":              TokLoopToEnactment,
	"loop_to_storyline":              TokLoopToStoryline,
	"loop_to_start_enactment":        TokLoopToStartEnactment,
	"loop_to_start_storyline":        TokLoopToStartStoryline,
	"up_to":                          TokUpTo,
	"times":                          TokTimes,
	"persist_scope":                  TokPersistScope,
	"trigger_priority":               TokTriggerPriority,
	"conditional_route":              TokConditionalRoute,
	"route":                          TokRoute,
	"has_badge":                      TokHasBadge,
	"not_has_badge":                  TokNotHasBadge,
	"has_tag":                        TokHasTag,
	"not_has_tag":                    TokNotHasTag,
	"if":                             TokIf,
	"else_if":                        TokElseIf,
	"and":                            TokAnd,
	"or":                             TokOr,
	"not":                            TokNot,
	"for":                            TokFor,
	"in":                             TokIn,
	"range":                          TokRange,
	"while":                          TokWhile,
	"break":                          TokBreak,
	"continue":                       TokContinue,
	"default":                        TokDefault,
	"sender":                         TokSender,
	"links":                          TokLinks,
	"pattern":                        TokPattern,
	"policy":                         TokPolicy,
	"use":                            TokUse,
	"as":                             TokAs,
	"scenes":                         TokScenes,
	"true":                           TokTrue,
	"data":                           TokData,
	"scene_defaults":                 TokSceneDefaults,
	"enactment_defaults":             TokEnactmentDefaults,
	"funnel":                         TokFunnel,
	"stage":                          TokStage,
	"page":                           TokPage,
	"block":                          TokBlock,
	"form":                           TokForm,
	"field":                          TokField,
	"path":                           TokPath,
	"domain":                         TokDomain,
	"ai":                             TokAI,
	"context":                        TokContext,
	"global":                         TokGlobal,
	"extend":                         TokExtend,
	"length":                         TokLength,
	"prompt":                         TokPrompt,
	"submit":                         TokSubmit,
	"abandon":                        TokAbandon,
	"purchase":                       TokPurchase,
	"product_id":                     TokProductId,
	"type":                           TokType,
	"checkout":                       TokCheckout,
	"upsell":                         TokUpsell,
	"lead_capture":                   TokLeadCapture,
	"required":                       TokRequired,
	"new":                            TokNew,
	"pdf":                            TokPdf,
	"lead_magnet":                    TokLeadMagnet,
	"reference":                      TokReference,
	"jump_to_stage":                  TokJumpToStage,
	"start_story":                    TokStartStory,
	"send_email":                     TokSendEmailAction,
	"redirect":                       TokRedirect,
	"provide_download":               TokProvideDownload,
	"must_have_badge":                TokMustHaveBadge,
	"must_not_have_badge":            TokMustNotHaveBadge,
	"watch":                          TokWatch,
	"video":                          TokVideo,
	"autoplay":                       TokAutoplay,
	"source_url":                     TokSourceURL,
	"false":                          TokFalse,
	"nil":                            TokNil,
	"product":                        TokProduct,
	"offer":                          TokOffer,
	"pricing_model":                  TokPricingModel,
	"price":                          TokPrice,
	"currency":                       TokCurrency,
	"includes_product":               TokIncludesProduct,
	"grants_badge":                   TokGrantsBadge,
	"order_bump":                     TokOrderBump,
	"one_click_upsell":               TokOneClickUpsell,
	"decline":                        TokDecline,
	"custom":                         TokCustom,
	"quiz":                           TokQuiz,
	"question":                       TokQuestion,
	"answer":                         TokAnswer,
	"add_score":                      TokAddScore,
	"score":                          TokScore,
	"site":                           TokSite,
	"seo":                            TokSEO,
	"navigation":                     TokNavigation,
	"header":                         TokHeader,
	"footer":                         TokFooter,
	"theme":                          TokTheme,
	"title":                          TokTitle,
	"description":                    TokDescription,
	"module":                         TokModule,
	"lesson":                         TokLesson,
	"video_url":                      TokVideoURL,
	"content":                        TokContent,
	"draft":                          TokDraft,
	"instructor":                     TokInstructor,
	"duration":                       TokDurationKw,
	"is_free":                        TokIsFree,
	"is_draft":                       TokIsDraft,
	"drip_days":                      TokDripDays,
	"content_gen":                    TokContentGen,
	"description_gen":                TokDescriptionGen,
	"pass_threshold":                 TokPassThreshold,
	"max_attempts":                   TokMaxAttempts,
	"options":                        TokOptions,
	"multiple_choice":                TokMultipleChoice,
	"short_answer":                   TokShortAnswer,
	"certificate":                    TokCertificate,
	"course_ref":                     TokCourseRef,
	"course":                         TokCourse,
	"media":                          TokMedia,
	"player_preset":                  TokPlayerPreset,
	"channel":                        TokChannel,
	"chapter":                        TokChapter,
	"turnstile":                      TokTurnstile,
	"cta":                            TokCTA,
	"annotation":                     TokAnnotation,
	"badge_rule":                     TokBadgeRule,
	"media_webhook":                  TokMediaWebhook,
	"media_ref":                      TokMediaRef,
	"start_sec":                      TokStartSec,
	"end_sec":                        TokEndSec,
	"threshold":                      TokThreshold,
	"operator":                       TokOperator,
	"enabled":                        TokEnabled,
	"progress":                       TokProgress,
	"complete":                       TokComplete,
	"play":                           TokPlay,
	"pause":                          TokPause,
	"rewatch":                        TokRewatch,
	"poster_url":                     TokPosterURL,
	"player_color":                   TokPlayerColor,
}

// LookupIdent returns the keyword TokenKind for ident, or TokIdent if it is
// not a keyword.
func LookupIdent(ident string) TokenKind {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokIdent
}
