package scripting

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// TestParseMediaDecl tests parsing of media declarations.
func TestParseMediaDecl(t *testing.T) {
	src := `
media "Intro Video" {
	title "Welcome Video"
	description "An introduction"
	source_url "https://storage.googleapis.com/sendhero-videos/VjdBaIC57zk-ufo.webm"
	poster_url "https://storage.googleapis.com/sendhero-videos/poster.jpg"
	tags "intro" "welcome"

	chapter "Welcome" {
		start_sec 0
		end_sec 30
	}

	chapter "Overview" {
		start_sec 30
		end_sec 120
	}

	turnstile {
		start_sec 15
		required
		field email
		field first_name
	}

	cta {
		start_sec 60
		text "Get the Course"
		url "https://example.com/offer"
		button_text "Enroll Now"
	}

	annotation {
		start_sec 45
		text "Key concept"
	}

	badge_rule {
		event progress
		operator ">="
		threshold 75
		badge "engaged"
	}

	badge_rule {
		event complete
		badge "finished"
	}
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	ast := pr.AST
	if len(ast.MediaDecls) != 1 {
		t.Fatalf("expected 1 media declaration, got %d", len(ast.MediaDecls))
	}

	m := ast.MediaDecls[0]
	if m.Name != "Intro Video" {
		t.Errorf("expected media name 'Intro Video', got %q", m.Name)
	}
	if m.Title != "Welcome Video" {
		t.Errorf("expected title 'Welcome Video', got %q", m.Title)
	}
	if m.SourceURL != "https://storage.googleapis.com/sendhero-videos/VjdBaIC57zk-ufo.webm" {
		t.Errorf("unexpected source_url: %q", m.SourceURL)
	}
	if m.PosterURL != "https://storage.googleapis.com/sendhero-videos/poster.jpg" {
		t.Errorf("unexpected poster_url: %q", m.PosterURL)
	}
	if len(m.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(m.Tags))
	}
	if len(m.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(m.Chapters))
	}
	if m.Chapters[0].Title != "Welcome" {
		t.Errorf("expected chapter title 'Welcome', got %q", m.Chapters[0].Title)
	}
	if m.Chapters[0].StartSec != 0 || m.Chapters[0].EndSec != 30 {
		t.Errorf("unexpected chapter timecodes: %d-%d", m.Chapters[0].StartSec, m.Chapters[0].EndSec)
	}
	if len(m.Interactions) != 3 {
		t.Errorf("expected 3 interactions, got %d", len(m.Interactions))
	}
	if m.Interactions[0].Kind != "turnstile" {
		t.Errorf("expected first interaction to be turnstile, got %q", m.Interactions[0].Kind)
	}
	if !m.Interactions[0].Required {
		t.Errorf("expected turnstile to be required")
	}
	if m.Interactions[1].Kind != "cta" {
		t.Errorf("expected second interaction to be cta, got %q", m.Interactions[1].Kind)
	}
	if m.Interactions[2].Kind != "annotation" {
		t.Errorf("expected third interaction to be annotation, got %q", m.Interactions[2].Kind)
	}
	if len(m.BadgeRules) != 2 {
		t.Errorf("expected 2 badge rules, got %d", len(m.BadgeRules))
	}
	if m.BadgeRules[0].EventName != "progress" {
		t.Errorf("expected first badge rule event 'progress', got %q", m.BadgeRules[0].EventName)
	}
	if m.BadgeRules[0].Threshold != 75 {
		t.Errorf("expected threshold 75, got %d", m.BadgeRules[0].Threshold)
	}
	if m.BadgeRules[0].BadgeName != "engaged" {
		t.Errorf("expected badge name 'engaged', got %q", m.BadgeRules[0].BadgeName)
	}
	if m.BadgeRules[1].EventName != "complete" {
		t.Errorf("expected second badge rule event 'complete', got %q", m.BadgeRules[1].EventName)
	}
}

// TestParsePlayerPresetDecl tests parsing of player_preset declarations.
func TestParsePlayerPresetDecl(t *testing.T) {
	src := `
player_preset "Brand Player" {
	player_color "#3b82f6"
	show_controls true
	show_big_play_button true
	allow_fullscreen true
	allow_playback_rate true
	autoplay false
	end_behavior "stop"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(pr.AST.PlayerPresets) != 1 {
		t.Fatalf("expected 1 player preset, got %d", len(pr.AST.PlayerPresets))
	}

	pp := pr.AST.PlayerPresets[0]
	if pp.Name != "Brand Player" {
		t.Errorf("expected name 'Brand Player', got %q", pp.Name)
	}
	if pp.PlayerColor != "#3b82f6" {
		t.Errorf("expected player_color '#3b82f6', got %q", pp.PlayerColor)
	}
	if !pp.ShowControls {
		t.Errorf("expected show_controls to be true")
	}
	if !pp.ShowBigPlayButton {
		t.Errorf("expected show_big_play_button to be true")
	}
	if !pp.AllowFullscreen {
		t.Errorf("expected allow_fullscreen to be true")
	}
	if pp.Autoplay {
		t.Errorf("expected autoplay to be false")
	}
	if pp.EndBehavior != "stop" {
		t.Errorf("expected end_behavior 'stop', got %q", pp.EndBehavior)
	}
}

// TestParseChannelDecl tests parsing of channel declarations.
func TestParseChannelDecl(t *testing.T) {
	src := `
channel "Course Playlist" {
	title "Complete Course"
	description "All videos"
	layout "playlist_right"
	items "Video 1" "Video 2" "Video 3"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(pr.AST.ChannelDecls) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(pr.AST.ChannelDecls))
	}

	ch := pr.AST.ChannelDecls[0]
	if ch.Title != "Complete Course" {
		t.Errorf("expected title 'Complete Course', got %q", ch.Title)
	}
	if len(ch.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(ch.Items))
	}
}

// TestParseMediaWebhookDecl tests parsing of media_webhook declarations.
func TestParseMediaWebhookDecl(t *testing.T) {
	src := `
media_webhook "Analytics Hook" {
	url "https://hooks.example.com/events"
	event_types "play" "complete" "turnstile_submit"
	enabled true
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(pr.AST.MediaWebhookDecls) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(pr.AST.MediaWebhookDecls))
	}

	wh := pr.AST.MediaWebhookDecls[0]
	if wh.URL != "https://hooks.example.com/events" {
		t.Errorf("unexpected url: %q", wh.URL)
	}
	if len(wh.EventTypes) != 3 {
		t.Errorf("expected 3 event types, got %d", len(wh.EventTypes))
	}
	if !wh.Enabled {
		t.Errorf("expected enabled to be true")
	}
}

// TestCompileMediaDecl tests that media declarations compile into Media entities.
func TestCompileMediaDecl(t *testing.T) {
	src := `
media "Sales Video" {
	title "Our Product Demo"
	description "Watch this demo"
	source_url "https://storage.googleapis.com/sendhero-videos/demo.mp4"

	chapter "Intro" {
		start_sec 0
		end_sec 60
	}

	badge_rule {
		event progress
		operator ">="
		threshold 50
		badge "half_watched"
	}

	badge_rule {
		event complete
		badge "video_complete"
	}
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	if len(result.MediaEntities) != 1 {
		t.Fatalf("expected 1 media entity, got %d", len(result.MediaEntities))
	}

	media := result.MediaEntities[0]
	if media.Title != "Our Product Demo" {
		t.Errorf("expected title 'Our Product Demo', got %q", media.Title)
	}
	if media.SourceURL != "https://storage.googleapis.com/sendhero-videos/demo.mp4" {
		t.Errorf("unexpected source_url: %q", media.SourceURL)
	}
	if media.Kind != "video" {
		t.Errorf("expected kind 'video', got %q", media.Kind)
	}
	if len(media.Chapters) != 1 {
		t.Errorf("expected 1 chapter, got %d", len(media.Chapters))
	}
	if len(media.BadgeRules) != 2 {
		t.Errorf("expected 2 badge rules, got %d", len(media.BadgeRules))
	}

	// Verify badges were pre-created
	if _, ok := result.Badges["half_watched"]; !ok {
		t.Errorf("expected badge 'half_watched' to be pre-created")
	}
	if _, ok := result.Badges["video_complete"]; !ok {
		t.Errorf("expected badge 'video_complete' to be pre-created")
	}
}

// TestCompilePlayerPresetDecl tests that player_preset declarations compile.
func TestCompilePlayerPresetDecl(t *testing.T) {
	src := `
player_preset "Dark Theme" {
	player_color "#1f2937"
	show_controls true
	autoplay false
	end_behavior "loop"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	if len(result.PlayerPresets) != 1 {
		t.Fatalf("expected 1 player preset, got %d", len(result.PlayerPresets))
	}

	preset := result.PlayerPresets[0]
	if preset.Name != "Dark Theme" {
		t.Errorf("expected name 'Dark Theme', got %q", preset.Name)
	}
	if preset.PlayerColor != "#1f2937" {
		t.Errorf("expected player_color '#1f2937', got %q", preset.PlayerColor)
	}
	if preset.Autoplay {
		t.Errorf("expected autoplay false")
	}
	if preset.EndBehavior != "loop" {
		t.Errorf("expected end_behavior 'loop', got %q", preset.EndBehavior)
	}
}

// TestVideoBlockWithMediaRef tests that video blocks can reference media entities.
func TestVideoBlockWithMediaRef(t *testing.T) {
	src := `
media "Demo Video" {
	title "Product Demo"
	source_url "https://example.com/demo.mp4"
}

player_preset "Custom" {
	player_color "#ff0000"
}

funnel "Sales Funnel" {
	domain "sales.example.com"

	route "Main" {
		order 1

		stage "Watch" {
			path "/watch"

			page "Video Page" {
				block "hero" {
					type video
					media_ref "Demo Video"
					player_preset "Custom"
				}
			}

			on watch "hero" >= 50 {
				do give_badge "engaged"
			}

			on progress "hero" >= 90 {
				do give_badge "almost_done"
			}

			on complete "hero" {
				do give_badge "completed"
			}
		}
	}
}

story "Followup" {
	storyline "Engaged" {
		enactment "Thanks" {
			scene "Email" {
				subject "Thanks for watching!"
				body "<p>Great work.</p>"
				from_email "a@b.com"
				from_name "Team"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	// Verify media entity was compiled
	if len(result.MediaEntities) != 1 {
		t.Fatalf("expected 1 media entity, got %d", len(result.MediaEntities))
	}

	// Verify player preset was compiled
	if len(result.PlayerPresets) != 1 {
		t.Fatalf("expected 1 player preset, got %d", len(result.PlayerPresets))
	}

	// Verify funnel was compiled with video block
	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}

	funnel := result.Funnels[0]
	if len(funnel.Routes) == 0 {
		t.Fatalf("expected at least 1 route")
	}
	stage := funnel.Routes[0].Stages[0]
	if len(stage.Pages) == 0 {
		t.Fatalf("expected at least 1 page")
	}
	page := stage.Pages[0]
	if len(page.Blocks) == 0 {
		t.Fatalf("expected at least 1 block")
	}
	block := page.Blocks[0]
	if block.BlockType != "video" {
		t.Errorf("expected block type 'video', got %q", block.BlockType)
	}
	if block.MediaPublicId != "Demo Video" {
		t.Errorf("expected media_ref 'Demo Video', got %q", block.MediaPublicId)
	}
	if block.PlayerPresetId != "Custom" {
		t.Errorf("expected player_preset 'Custom', got %q", block.PlayerPresetId)
	}

	// Verify badges were created
	expectedBadges := []string{"engaged", "almost_done", "completed"}
	for _, name := range expectedBadges {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q to exist", name)
		}
	}
}

// TestVideoTriggerTypes tests that all video trigger types parse and compile.
func TestVideoTriggerTypes(t *testing.T) {
	src := `
funnel "Video Triggers" {
	domain "triggers.example.com"

	route "Main" {
		order 1

		stage "Watch" {
			path "/watch"

			page "Video" {
				block "video" {
					type video
					source_url "https://example.com/video.mp4"
				}
			}

			on play "video" {
				do give_badge "played"
			}

			on pause "video" {
				do give_badge "paused"
			}

			on progress "video" >= 50 {
				do give_badge "half"
			}

			on complete "video" {
				do give_badge "done"
			}

			on rewatch "video" {
				do give_badge "rewatcher"
			}

			on watch "video" > 75 {
				do give_badge "watched_75"
			}
		}
	}
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	expectedBadges := []string{"played", "paused", "half", "done", "rewatcher", "watched_75"}
	for _, name := range expectedBadges {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q to be created", name)
		}
	}
}

// TestLessonWithMediaRef tests lesson with media_ref fallback.
func TestLessonWithMediaRef(t *testing.T) {
	src := `
product "Course" {
	description "A video course"
	type "course"

	module "Module 1" {
		lesson "Intro" {
			media_ref "intro-video-id"
			content "<p>Watch the intro.</p>"
		}

		lesson "Basics" {
			video_url "https://example.com/basics.mp4"
			content "<p>Learn the basics.</p>"
		}
	}
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	if len(result.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(result.Products))
	}

	product := result.Products[0]
	if len(product.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(product.Modules))
	}

	module := product.Modules[0]
	if len(module.Lessons) != 2 {
		t.Fatalf("expected 2 lessons, got %d", len(module.Lessons))
	}

	// First lesson uses media_ref
	if module.Lessons[0].MediaPublicId != "intro-video-id" {
		t.Errorf("expected media_ref 'intro-video-id', got %q", module.Lessons[0].MediaPublicId)
	}
	if module.Lessons[0].VideoURL != "" {
		t.Errorf("expected video_url to be empty for media_ref lesson, got %q", module.Lessons[0].VideoURL)
	}

	// Second lesson uses legacy video_url
	if module.Lessons[1].VideoURL != "https://example.com/basics.mp4" {
		t.Errorf("expected video_url, got %q", module.Lessons[1].VideoURL)
	}
	if module.Lessons[1].MediaPublicId != "" {
		t.Errorf("expected media_ref to be empty for video_url lesson, got %q", module.Lessons[1].MediaPublicId)
	}
}

// TestCompileFullVideoIntelligenceFixture tests the full 52_video_intelligence.ss fixture.
func TestCompileFullVideoIntelligenceFixture(t *testing.T) {
	result := CompileScript(IntegrationTestVideoIntelligence, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	// Verify entity counts
	if len(result.Stories) != 1 {
		t.Errorf("expected 1 story, got %d", len(result.Stories))
	}
	if len(result.Funnels) != 1 {
		t.Errorf("expected 1 funnel, got %d", len(result.Funnels))
	}
	if len(result.Products) != 1 {
		t.Errorf("expected 1 product, got %d", len(result.Products))
	}
	if len(result.MediaEntities) != 2 {
		t.Errorf("expected 2 media entities, got %d", len(result.MediaEntities))
	}
	if len(result.PlayerPresets) != 1 {
		t.Errorf("expected 1 player preset, got %d", len(result.PlayerPresets))
	}
	if len(result.Channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(result.Channels))
	}
	if len(result.MediaWebhooks) != 1 {
		t.Errorf("expected 1 media webhook, got %d", len(result.MediaWebhooks))
	}

	// Verify badge count (should include all video-related badges)
	expectedBadges := []string{
		"video_started", "video_engaged", "video_completed",
		"advanced_complete",
		"engaged_viewer", "almost_done", "watched_full",
		"video_lead", "customer",
	}
	for _, name := range expectedBadges {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q to be created", name)
		}
	}

	// Verify media entities have correct structure
	introMedia := result.MediaEntities[0]
	if introMedia.Title != "Welcome to Our Course" {
		t.Errorf("unexpected title: %q", introMedia.Title)
	}
	if len(introMedia.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(introMedia.Chapters))
	}
	if len(introMedia.Interactions) != 3 {
		t.Errorf("expected 3 interactions, got %d", len(introMedia.Interactions))
	}
	if len(introMedia.BadgeRules) != 3 {
		t.Errorf("expected 3 badge rules, got %d", len(introMedia.BadgeRules))
	}

	// Verify channel has items
	channel := result.Channels[0]
	if len(channel.Items) != 2 {
		t.Errorf("expected 2 channel items, got %d", len(channel.Items))
	}

	// Verify webhook
	webhook := result.MediaWebhooks[0]
	if webhook.Name != "Analytics Hook" {
		t.Errorf("unexpected webhook name: %q", webhook.Name)
	}
	if len(webhook.EventTypes) != 3 {
		t.Errorf("expected 3 event types, got %d", len(webhook.EventTypes))
	}

	// Verify product has lessons with media_ref
	product := result.Products[0]
	if len(product.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(product.Modules))
	}
	lessons := product.Modules[0].Lessons
	if len(lessons) != 2 {
		t.Fatalf("expected 2 lessons, got %d", len(lessons))
	}
	if lessons[0].MediaPublicId != "Intro Video" {
		t.Errorf("expected lesson media_ref 'Intro Video', got %q", lessons[0].MediaPublicId)
	}

	t.Logf("Full video intelligence fixture compiled successfully:")
	t.Logf("  Stories: %d", len(result.Stories))
	t.Logf("  Funnels: %d", len(result.Funnels))
	t.Logf("  Products: %d", len(result.Products))
	t.Logf("  Media: %d", len(result.MediaEntities))
	t.Logf("  PlayerPresets: %d", len(result.PlayerPresets))
	t.Logf("  Channels: %d", len(result.Channels))
	t.Logf("  Webhooks: %d", len(result.MediaWebhooks))
	t.Logf("  Badges: %d", len(result.Badges))
}

// TestParsePlayerPresetAllFields tests parsing all player preset control fields.
func TestParsePlayerPresetAllFields(t *testing.T) {
	src := `
player_preset "Full Controls" {
	player_color "#ff5733"
	show_controls true
	show_rewind true
	show_fast_forward true
	show_skip true
	show_download true
	hide_progress_bar true
	show_big_play_button true
	allow_fullscreen true
	allow_playback_rate true
	allow_seeking true
	autoplay true
	muted_default true
	disable_pause true
	loop true
	rounded_player true
	end_behavior "prevent_replay"
	chapter_style "buttons"
	chapter_position "bottom-center"
	chapter_click_jump true
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(pr.AST.PlayerPresets) != 1 {
		t.Fatalf("expected 1 player preset, got %d", len(pr.AST.PlayerPresets))
	}

	pp := pr.AST.PlayerPresets[0]

	// Test all button visibility settings
	if !pp.ShowControls {
		t.Errorf("expected show_controls to be true")
	}
	if !pp.ShowRewind {
		t.Errorf("expected show_rewind to be true")
	}
	if !pp.ShowFastForward {
		t.Errorf("expected show_fast_forward to be true")
	}
	if !pp.ShowSkip {
		t.Errorf("expected show_skip to be true")
	}
	if !pp.ShowDownload {
		t.Errorf("expected show_download to be true")
	}
	if !pp.HideProgressBar {
		t.Errorf("expected hide_progress_bar to be true")
	}

	// Test other controls
	if !pp.ShowBigPlayButton {
		t.Errorf("expected show_big_play_button to be true")
	}
	if !pp.AllowFullscreen {
		t.Errorf("expected allow_fullscreen to be true")
	}
	if !pp.AllowPlaybackRate {
		t.Errorf("expected allow_playback_rate to be true")
	}
	if !pp.AllowSeeking {
		t.Errorf("expected allow_seeking to be true")
	}

	// Test behaviour
	if !pp.Autoplay {
		t.Errorf("expected autoplay to be true")
	}
	if !pp.MutedDefault {
		t.Errorf("expected muted_default to be true")
	}
	if !pp.DisablePause {
		t.Errorf("expected disable_pause to be true")
	}
	if !pp.Loop {
		t.Errorf("expected loop to be true")
	}

	// Test end behaviour and other settings
	if pp.EndBehavior != "prevent_replay" {
		t.Errorf("expected end_behavior 'prevent_replay', got %q", pp.EndBehavior)
	}
	if !pp.RoundedPlayer {
		t.Errorf("expected rounded_player to be true")
	}

	// Test chapter controls
	if pp.ChapterStyle != "buttons" {
		t.Errorf("expected chapter_style 'buttons', got %q", pp.ChapterStyle)
	}
	if pp.ChapterPosition != "bottom-center" {
		t.Errorf("expected chapter_position 'bottom-center', got %q", pp.ChapterPosition)
	}
	if !pp.ChapterClickJump {
		t.Errorf("expected chapter_click_jump to be true")
	}
}

// TestCompilePlayerPresetAllFields tests compiling all player preset fields.
func TestCompilePlayerPresetAllFields(t *testing.T) {
	src := `
player_preset "Full Controls" {
	player_color "#ff5733"
	show_controls true
	show_rewind true
	show_fast_forward true
	show_skip true
	show_download true
	hide_progress_bar true
	show_big_play_button true
	allow_fullscreen true
	allow_playback_rate true
	allow_seeking true
	autoplay true
	muted_default true
	disable_pause true
	end_behavior "prevent_replay"
	rounded_player true
	chapter_style "buttons"
	chapter_position "bottom-center"
	chapter_click_jump true
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	if len(result.PlayerPresets) != 1 {
		t.Fatalf("expected 1 player preset, got %d", len(result.PlayerPresets))
	}

	preset := result.PlayerPresets[0]

	// Verify all button visibility settings compiled
	if !preset.ShowControls {
		t.Errorf("expected ShowControls true, got %v", preset.ShowControls)
	}
	if !preset.ShowRewind {
		t.Errorf("expected ShowRewind true, got %v", preset.ShowRewind)
	}
	if !preset.ShowFastForward {
		t.Errorf("expected ShowFastForward true, got %v", preset.ShowFastForward)
	}
	if !preset.ShowSkip {
		t.Errorf("expected ShowSkip true, got %v", preset.ShowSkip)
	}
	if !preset.ShowDownload {
		t.Errorf("expected ShowDownload true, got %v", preset.ShowDownload)
	}
	if !preset.HideProgressBar {
		t.Errorf("expected HideProgressBar true, got %v", preset.HideProgressBar)
	}

	// Verify other controls compiled
	if !preset.ShowBigPlayButton {
		t.Errorf("expected ShowBigPlayButton true, got %v", preset.ShowBigPlayButton)
	}
	if !preset.AllowFullscreen {
		t.Errorf("expected AllowFullscreen true, got %v", preset.AllowFullscreen)
	}
	if !preset.AllowPlaybackRate {
		t.Errorf("expected AllowPlaybackRate true, got %v", preset.AllowPlaybackRate)
	}
	if !preset.AllowSeeking {
		t.Errorf("expected AllowSeeking true, got %v", preset.AllowSeeking)
	}

	// Verify behaviour compiled
	if !preset.Autoplay {
		t.Errorf("expected Autoplay true, got %v", preset.Autoplay)
	}
	if !preset.MutedDefault {
		t.Errorf("expected MutedDefault true, got %v", preset.MutedDefault)
	}
	if !preset.DisablePause {
		t.Errorf("expected DisablePause true, got %v", preset.DisablePause)
	}
	if preset.EndBehavior != "prevent_replay" {
		t.Errorf("expected EndBehavior 'prevent_replay', got %q", preset.EndBehavior)
	}
	if !preset.RoundedPlayer {
		t.Errorf("expected RoundedPlayer true, got %v", preset.RoundedPlayer)
	}

	// Verify chapter controls compiled
	if preset.ChapterStyle != "buttons" {
		t.Errorf("expected ChapterStyle 'buttons', got %q", preset.ChapterStyle)
	}
	if preset.ChapterPosition != "bottom-center" {
		t.Errorf("expected ChapterPosition 'bottom-center', got %q", preset.ChapterPosition)
	}
	if !preset.ChapterClickJump {
		t.Errorf("expected ChapterClickJump true, got %v", preset.ChapterClickJump)
	}
}

// TestPlayerPresetChapterPositions tests all chapter position values.
func TestPlayerPresetChapterPositions(t *testing.T) {
	positions := []string{
		"top-left", "top-center", "top-right",
		"center",
		"bottom-left", "bottom-center", "bottom-right",
	}

	for _, pos := range positions {
		src := `
player_preset "Chapter Pos" {
	chapter_position "` + pos + `"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
		result := CompileScript(src, "sub_123", bson.NewObjectId())
		if result.Diagnostics.HasErrors() {
			t.Fatalf("compilation failed for position %q: %v", pos, result.Diagnostics)
		}

		if len(result.PlayerPresets) != 1 {
			t.Fatalf("expected 1 preset for position %q", pos)
		}

		if result.PlayerPresets[0].ChapterPosition != pos {
			t.Errorf("position %q: expected %q, got %q", pos, pos, result.PlayerPresets[0].ChapterPosition)
		}
	}
}

// TestPlayerPresetChapterStyles tests chapter display style variations.
func TestPlayerPresetChapterStyles(t *testing.T) {
	styles := []string{"hover", "buttons"}

	for _, style := range styles {
		src := `
player_preset "Chapter Style" {
	chapter_style "` + style + `"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
		result := CompileScript(src, "sub_123", bson.NewObjectId())
		if result.Diagnostics.HasErrors() {
			t.Fatalf("compilation failed for style %q: %v", style, result.Diagnostics)
		}

		if len(result.PlayerPresets) != 1 {
			t.Fatalf("expected 1 preset for style %q", style)
		}

		if result.PlayerPresets[0].ChapterStyle != style {
			t.Errorf("style %q: expected %q, got %q", style, style, result.PlayerPresets[0].ChapterStyle)
		}
	}
}

// TestPlayerPresetEndBehaviors tests all end_behavior values.
func TestPlayerPresetEndBehaviors(t *testing.T) {
	behaviors := []string{"stop", "loop", "prevent_replay"}

	for _, behavior := range behaviors {
		src := `
player_preset "End Behavior" {
	end_behavior "` + behavior + `"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
		result := CompileScript(src, "sub_123", bson.NewObjectId())
		if result.Diagnostics.HasErrors() {
			t.Fatalf("compilation failed for behavior %q: %v", behavior, result.Diagnostics)
		}

		if len(result.PlayerPresets) != 1 {
			t.Fatalf("expected 1 preset for behavior %q", behavior)
		}

		if result.PlayerPresets[0].EndBehavior != behavior {
			t.Errorf("behavior %q: expected %q, got %q", behavior, behavior, result.PlayerPresets[0].EndBehavior)
		}
	}
}

// TestPlayerPresetMinimalSettings tests that defaults work correctly.
func TestPlayerPresetMinimalSettings(t *testing.T) {
	src := `
player_preset "Minimal" {
	player_color "#000000"
}

story "Minimal" {
	storyline "A" {
		enactment "B" {
			scene "C" {
				subject "test"
				body "test"
				from_email "a@b.com"
				from_name "A"
				reply_to "a@b.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	if len(result.PlayerPresets) != 1 {
		t.Fatalf("expected 1 player preset")
	}

	preset := result.PlayerPresets[0]

	// Verify defaults are false/empty
	if preset.ShowRewind {
		t.Errorf("expected ShowRewind to default to false")
	}
	if preset.ShowFastForward {
		t.Errorf("expected ShowFastForward to default to false")
	}
	if preset.ShowSkip {
		t.Errorf("expected ShowSkip to default to false")
	}
	if preset.ShowDownload {
		t.Errorf("expected ShowDownload to default to false")
	}
	if preset.HideProgressBar {
		t.Errorf("expected HideProgressBar to default to false")
	}
	if preset.DisablePause {
		t.Errorf("expected DisablePause to default to false")
	}
	if preset.ChapterStyle != "" {
		t.Errorf("expected ChapterStyle to be empty, got %q", preset.ChapterStyle)
	}
}
