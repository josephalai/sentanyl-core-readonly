package scripting

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// =====================================================================
// LMS Scripting Tests — parser and compiler
// =====================================================================

// ---------- Token Tests ----------

func TestLMSTokensRegistered(t *testing.T) {
	lmsKeywords := map[string]TokenKind{
		"course":          TokCourse,
		"instructor":      TokInstructor,
		"description_gen": TokDescriptionGen,
		"is_free":         TokIsFree,
		"is_draft":        TokIsDraft,
		"drip_days":       TokDripDays,
		"pass_threshold":  TokPassThreshold,
		"max_attempts":    TokMaxAttempts,
		"options":         TokOptions,
		"multiple_choice": TokMultipleChoice,
		"short_answer":    TokShortAnswer,
		"certificate":     TokCertificate,
		"course_ref":      TokCourseRef,
		"duration":        TokDurationKw,
	}
	for kw, expected := range lmsKeywords {
		actual := LookupIdent(kw)
		if actual != expected {
			t.Errorf("LookupIdent(%q) = %v, want %v", kw, actual, expected)
		}
	}
}

// ---------- Parser Tests ----------

func TestParseLMSCourseProduct(t *testing.T) {
	src := `
product "Go Fundamentals" {
	type course
	description "Learn Go from scratch"
	instructor "John Doe"

	module "Getting Started" {
		lesson "Hello World" {
			video_url "https://example.com/hello.mp4"
			content "<p>Welcome to Go!</p>"
			is_free true
		}
		lesson "Variables" {
			video_url "https://example.com/vars.mp4"
			content "<p>Variables in Go</p>"
			drip_days 7
			duration "01:30:00"
		}
	}

	module "Advanced" {
		lesson "Concurrency" {
			video_url "https://example.com/concurrency.mp4"
			content "<p>Goroutines and channels</p>"
			is_draft true
		}

		quiz "Module Quiz" {
			pass_threshold 70
			max_attempts 3
			question "What is a goroutine?" {
				type multiple_choice
				options ["A thread", "A lightweight thread", "A process", "A function"]
				answer 1
			}
			question "Explain channels" {
				type short_answer
				answer "typed conduit"
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
	pr := ParseScript(src); ast := pr.AST; diags := pr.Diagnostics
	if diags.HasErrors() {
		for _, d := range diags {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(ast.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(ast.Products))
	}
	p := ast.Products[0]
	if p.Name != "Go Fundamentals" {
		t.Errorf("product name = %q, want %q", p.Name, "Go Fundamentals")
	}
	if p.ProductType != "course" {
		t.Errorf("product type = %q, want %q", p.ProductType, "course")
	}
	if p.Instructor != "John Doe" {
		t.Errorf("instructor = %q, want %q", p.Instructor, "John Doe")
	}
	if len(p.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(p.Modules))
	}

	// Module 1
	m1 := p.Modules[0]
	if m1.Title != "Getting Started" {
		t.Errorf("module 1 title = %q, want %q", m1.Title, "Getting Started")
	}
	if len(m1.Lessons) != 2 {
		t.Fatalf("module 1: expected 2 lessons, got %d", len(m1.Lessons))
	}
	l1 := m1.Lessons[0]
	if !l1.IsFree {
		t.Error("lesson 1 should be is_free=true")
	}
	l2 := m1.Lessons[1]
	if l2.DripDays != 7 {
		t.Errorf("lesson 2 drip_days = %d, want 7", l2.DripDays)
	}
	if l2.Duration != "01:30:00" {
		t.Errorf("lesson 2 duration = %q, want %q", l2.Duration, "01:30:00")
	}

	// Module 2
	m2 := p.Modules[1]
	if len(m2.Lessons) != 1 {
		t.Fatalf("module 2: expected 1 lesson, got %d", len(m2.Lessons))
	}
	if !m2.Lessons[0].IsDraft {
		t.Error("module 2 lesson should be is_draft=true")
	}

	// Quiz
	if len(m2.Quizzes) != 1 {
		t.Fatalf("module 2: expected 1 quiz, got %d", len(m2.Quizzes))
	}
	q := m2.Quizzes[0]
	if q.Title != "Module Quiz" {
		t.Errorf("quiz title = %q, want %q", q.Title, "Module Quiz")
	}
	if q.PassThreshold != 70 {
		t.Errorf("pass_threshold = %d, want 70", q.PassThreshold)
	}
	if q.MaxAttempts != 3 {
		t.Errorf("max_attempts = %d, want 3", q.MaxAttempts)
	}
	if len(q.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(q.Questions))
	}
	if q.Questions[0].Type != "multiple_choice" {
		t.Errorf("q1 type = %q, want multiple_choice", q.Questions[0].Type)
	}
	if len(q.Questions[0].Options) != 4 {
		t.Errorf("q1 options count = %d, want 4", len(q.Questions[0].Options))
	}
	if q.Questions[0].Answer != 1 {
		t.Errorf("q1 answer = %v, want 1", q.Questions[0].Answer)
	}
	if q.Questions[1].Type != "short_answer" {
		t.Errorf("q2 type = %q, want short_answer", q.Questions[1].Type)
	}
	if q.Questions[1].Answer != "typed conduit" {
		t.Errorf("q2 answer = %v, want %q", q.Questions[1].Answer, "typed conduit")
	}
}

func TestParseLMSContentGen(t *testing.T) {
	src := `
product "AI Course" {
	type course
	description "Test"

	module "Intro" {
		lesson "Lesson 1" {
			description_gen {
				instruction "Generate intro lesson content about AI basics"
				references ["https://example.com/ai-basics"]
				theme "modern"
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
	pr := ParseScript(src); ast := pr.AST; diags := pr.Diagnostics
	if diags.HasErrors() {
		for _, d := range diags {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(ast.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(ast.Products))
	}
	lesson := ast.Products[0].Modules[0].Lessons[0]
	if lesson.ContentGen == nil {
		t.Fatal("expected ContentGen to be set")
	}
	if lesson.ContentGen.Instruction != "Generate intro lesson content about AI basics" {
		t.Errorf("instruction = %q", lesson.ContentGen.Instruction)
	}
	if len(lesson.ContentGen.References) != 1 {
		t.Errorf("references count = %d, want 1", len(lesson.ContentGen.References))
	}
	if lesson.ContentGen.Theme != "modern" {
		t.Errorf("theme = %q, want %q", lesson.ContentGen.Theme, "modern")
	}
}

func TestParseDescriptionGen(t *testing.T) {
	src := `
product "Smart Course" {
	type course
	description_gen {
		instruction "Create a compelling course description"
		references ["https://example.com/ref1"]
	}

	module "Module 1" {
		lesson "L1" {
			content "<p>Hello</p>"
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
	pr := ParseScript(src); ast := pr.AST; diags := pr.Diagnostics
	if diags.HasErrors() {
		for _, d := range diags {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	if len(ast.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(ast.Products))
	}
	p := ast.Products[0]
	if p.DescriptionGen == nil {
		t.Fatal("expected DescriptionGen to be set")
	}
	if p.DescriptionGen.Instruction != "Create a compelling course description" {
		t.Errorf("instruction = %q", p.DescriptionGen.Instruction)
	}
}

// ---------- Compiler Tests ----------

func TestCompileLMSCourseProduct(t *testing.T) {
	src := `
product "Docker Academy" {
	type course
	description "Learn Docker from zero to hero"
	instructor "Jane Smith"

	module "Docker Basics" {
		lesson "What is Docker?" {
			video_url "https://example.com/docker-intro.mp4"
			content "<p>Docker is a containerization platform.</p>"
			is_free true
		}
		lesson "Containers vs VMs" {
			video_url "https://example.com/containers.mp4"
			content "<p>Understanding the difference.</p>"
			drip_days 3
		}
	}

	module "Advanced Docker" {
		lesson "Docker Compose" {
			video_url "https://example.com/compose.mp4"
			content "<p>Multi-container applications.</p>"
			duration "00:45:00"
		}

		quiz "Docker Quiz" {
			pass_threshold 80
			max_attempts 2
			question "What is a Dockerfile?" {
				type multiple_choice
				options ["A config file", "A build script", "Both A and B", "None"]
				answer 2
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

	if len(result.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(result.Products))
	}

	product := result.Products[0]

	// Verify basic fields
	if product.Name != "Docker Academy" {
		t.Errorf("name = %q, want %q", product.Name, "Docker Academy")
	}
	if product.ProductType != "course" {
		t.Errorf("product_type = %q, want %q", product.ProductType, "course")
	}
	if product.InstructorName != "Jane Smith" {
		t.Errorf("instructor_name = %q, want %q", product.InstructorName, "Jane Smith")
	}
	if product.Status != "published" {
		t.Errorf("status = %q, want %q", product.Status, "published")
	}

	// Verify CourseModules
	if len(product.CourseModules) != 2 {
		t.Fatalf("expected 2 course_modules, got %d", len(product.CourseModules))
	}

	cm1 := product.CourseModules[0]
	if cm1.Title != "Docker Basics" {
		t.Errorf("module 1 title = %q", cm1.Title)
	}
	if cm1.Order != 1 {
		t.Errorf("module 1 order = %d, want 1", cm1.Order)
	}
	if len(cm1.Lessons) != 2 {
		t.Fatalf("module 1: expected 2 lessons, got %d", len(cm1.Lessons))
	}
	if !cm1.Lessons[0].IsFree {
		t.Error("module 1 lesson 1 should be is_free")
	}
	if cm1.Lessons[1].DripDays != 3 {
		t.Errorf("module 1 lesson 2 drip_days = %d, want 3", cm1.Lessons[1].DripDays)
	}

	cm2 := product.CourseModules[1]
	if cm2.Title != "Advanced Docker" {
		t.Errorf("module 2 title = %q", cm2.Title)
	}
	if len(cm2.Lessons) != 1 {
		t.Fatalf("module 2: expected 1 lesson, got %d", len(cm2.Lessons))
	}
	if cm2.Lessons[0].Duration != "00:45:00" {
		t.Errorf("module 2 lesson duration = %q, want %q", cm2.Lessons[0].Duration, "00:45:00")
	}
	if cm2.QuizSlug == "" {
		t.Error("module 2 should have quiz_slug set")
	}

	// Verify TotalLessons
	if product.TotalLessons != 3 {
		t.Errorf("total_lessons = %d, want 3", product.TotalLessons)
	}

	// Verify legacy Modules are also populated (backward compatibility)
	if len(product.Modules) != 2 {
		t.Errorf("legacy modules count = %d, want 2", len(product.Modules))
	}

	// Verify quizzes
	if len(result.Quizzes) != 1 {
		t.Fatalf("expected 1 quiz, got %d", len(result.Quizzes))
	}
	quiz := result.Quizzes[0]
	if quiz.Title != "Docker Quiz" {
		t.Errorf("quiz title = %q", quiz.Title)
	}
	if quiz.PassThreshold != 80 {
		t.Errorf("pass_threshold = %d, want 80", quiz.PassThreshold)
	}
	if quiz.MaxAttempts != 2 {
		t.Errorf("max_attempts = %d, want 2", quiz.MaxAttempts)
	}
	if len(quiz.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(quiz.Questions))
	}
	if quiz.Questions[0].CorrectAnswer != 2 {
		t.Errorf("correct_answer = %d, want 2", quiz.Questions[0].CorrectAnswer)
	}
	if quiz.ProductID != product.Id {
		t.Errorf("quiz product_id = %v, want %v", quiz.ProductID, product.Id)
	}
	if quiz.ModuleSlug == "" {
		t.Error("quiz module_slug should be set")
	}
}

func TestCompileLMSContentGenPending(t *testing.T) {
	src := `
product "Gen Course" {
	type course
	description_gen {
		instruction "Auto-describe this course"
	}

	module "M1" {
		lesson "L1" {
			description_gen {
				instruction "Generate lesson content"
				theme "academic"
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

	product := result.Products[0]

	// Verify description_gen on product
	if product.DescriptionGenStatus != "pending" {
		t.Errorf("description_gen_status = %q, want %q", product.DescriptionGenStatus, "pending")
	}
	if product.DescriptionGenConfig == nil {
		t.Fatal("expected description_gen_config to be set")
	}
	if product.DescriptionGenConfig.Instruction != "Auto-describe this course" {
		t.Errorf("description gen instruction = %q", product.DescriptionGenConfig.Instruction)
	}

	// Verify content_gen on lesson
	if len(product.CourseModules) != 1 {
		t.Fatalf("expected 1 course module, got %d", len(product.CourseModules))
	}
	lesson := product.CourseModules[0].Lessons[0]
	if lesson.ContentGenStatus != "pending" {
		t.Errorf("content_gen_status = %q, want %q", lesson.ContentGenStatus, "pending")
	}
	if lesson.ContentGenConfig == nil {
		t.Fatal("expected content_gen_config on lesson")
	}
	if lesson.ContentGenConfig.Theme != "academic" {
		t.Errorf("theme = %q, want %q", lesson.ContentGenConfig.Theme, "academic")
	}
}

func TestCompileNonCourseProductUnchanged(t *testing.T) {
	src := `
product "eBook" {
	type download
	description "A digital download"
	module "Chapter 1" {
		lesson "Introduction" {
			content "<p>Intro</p>"
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

	product := result.Products[0]
	// Non-course products should use legacy Modules
	if len(product.Modules) != 1 {
		t.Errorf("legacy modules count = %d, want 1", len(product.Modules))
	}
	// CourseModules should be nil for non-course products
	if len(product.CourseModules) != 0 {
		t.Errorf("course_modules count = %d, want 0 for non-course product", len(product.CourseModules))
	}
	// No quizzes should be generated
	if len(result.Quizzes) != 0 {
		t.Errorf("quizzes count = %d, want 0 for non-course product", len(result.Quizzes))
	}
}

func TestParseLMSQuizSlugs(t *testing.T) {
	src := `
product "Course" {
	type course
	module "M1" "module-one" {
		lesson "L1" "lesson-one" {
			content "<p>Hello</p>"
		}
		quiz "Quiz 1" "quiz-one" {
			pass_threshold 50
			question "Q1" "q-one" {
				type multiple_choice
				options ["A", "B"]
				answer 0
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
	pr := ParseScript(src); ast := pr.AST; diags := pr.Diagnostics
	if diags.HasErrors() {
		for _, d := range diags {
			t.Errorf("parse error: %v", d)
		}
		t.FailNow()
	}

	mod := ast.Products[0].Modules[0]
	if mod.Slug != "module-one" {
		t.Errorf("module slug = %q, want %q", mod.Slug, "module-one")
	}
	lesson := mod.Lessons[0]
	if lesson.Slug != "lesson-one" {
		t.Errorf("lesson slug = %q, want %q", lesson.Slug, "lesson-one")
	}
	quiz := mod.Quizzes[0]
	if quiz.Slug != "quiz-one" {
		t.Errorf("quiz slug = %q, want %q", quiz.Slug, "quiz-one")
	}
	q := quiz.Questions[0]
	if q.Slug != "q-one" {
		t.Errorf("question slug = %q, want %q", q.Slug, "q-one")
	}
}

func TestCompileMultipleQuizzes(t *testing.T) {
	src := `
product "Multi Quiz Course" {
	type course

	module "Mod A" {
		lesson "L1" {
			content "Hello"
		}
		quiz "Quiz A" {
			pass_threshold 60
			question "Q1" {
				type multiple_choice
				options ["A", "B"]
				answer 0
			}
		}
	}

	module "Mod B" {
		lesson "L2" {
			content "World"
		}
		quiz "Quiz B" {
			pass_threshold 80
			question "Q1" {
				type short_answer
				answer "hello world"
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

	if len(result.Quizzes) != 2 {
		t.Fatalf("expected 2 quizzes, got %d", len(result.Quizzes))
	}

	if result.Quizzes[0].PassThreshold != 60 {
		t.Errorf("quiz A pass_threshold = %d, want 60", result.Quizzes[0].PassThreshold)
	}
	if result.Quizzes[1].PassThreshold != 80 {
		t.Errorf("quiz B pass_threshold = %d, want 80", result.Quizzes[1].PassThreshold)
	}

	// Each quiz should be linked to its module
	if result.Quizzes[0].ModuleSlug == "" {
		t.Error("quiz A should have module_slug")
	}
	if result.Quizzes[1].ModuleSlug == "" {
		t.Error("quiz B should have module_slug")
	}
	if result.Quizzes[0].ModuleSlug == result.Quizzes[1].ModuleSlug {
		t.Error("quizzes should have different module_slugs")
	}
}

// TestCompileFullLMSFixture tests a full LMS fixture through the entire pipeline.
func TestCompileFullLMSFixture(t *testing.T) {
	src := `
product "Sentanyl Academy" {
	type course
	description "The complete Sentanyl platform course"
	instructor "Platform Team"

	description_gen {
		instruction "Generate a professional course description"
		references ["https://sentanyl.com/docs"]
	}

	module "Introduction" {
		lesson "Welcome" {
			video_url "https://example.com/welcome.mp4"
			content "<p>Welcome to the course!</p>"
			is_free true
			duration "00:10:00"
		}
		lesson "Platform Overview" {
			media_ref "overview-video-id"
			description_gen {
				instruction "Generate overview content"
				theme "professional"
			}
			drip_days 0
		}
	}

	module "Core Concepts" {
		lesson "Stories & Storylines" {
			video_url "https://example.com/stories.mp4"
			content "<p>Understanding the story model</p>"
			drip_days 7
		}
		lesson "Funnels & Pages" {
			video_url "https://example.com/funnels.mp4"
			content "<p>Building sales funnels</p>"
			is_draft true
		}

		quiz "Core Concepts Quiz" {
			pass_threshold 75
			max_attempts 3
			question "What is a storyline?" {
				type multiple_choice
				options ["A sequence of events", "A narrative arc", "An automation pipeline", "All of the above"]
				answer 3
			}
			question "Describe a funnel stage" {
				type short_answer
				answer "A step in the conversion funnel"
			}
		}
	}

	module "Advanced Topics" {
		lesson "Video Intelligence" {
			video_url "https://example.com/video-intel.mp4"
			content "<p>Analytics and engagement tracking</p>"
		}

		quiz "Advanced Quiz" {
			pass_threshold 80
			question "What does media_ref do?" {
				type multiple_choice
				options ["References a media asset", "Creates a new video", "Deletes media", "None"]
				answer 0
			}
		}
	}
}

offer "Academy Access" {
	pricing_model one_time
	price 297.00
	currency "usd"
	includes_product "Sentanyl Academy"
	grants_badge "academy_enrolled"
}

story "Academy Completion" {
	storyline "Welcome" {
		enactment "Day 1" {
			scene "Welcome Email" {
				subject "Welcome to Sentanyl Academy!"
				body "You are now enrolled."
				from_email "academy@sentanyl.com"
				from_name "Sentanyl Academy"
				reply_to "support@sentanyl.com"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_academy", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("compile error: %v", d)
		}
		t.FailNow()
	}

	// Verify all entities compiled
	if len(result.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(result.Products))
	}
	if len(result.Offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(result.Offers))
	}
	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	if len(result.Quizzes) != 2 {
		t.Fatalf("expected 2 quizzes, got %d", len(result.Quizzes))
	}

	product := result.Products[0]

	// Verify course structure
	if len(product.CourseModules) != 3 {
		t.Fatalf("expected 3 course_modules, got %d", len(product.CourseModules))
	}
	if product.TotalLessons != 5 {
		t.Errorf("total_lessons = %d, want 5", product.TotalLessons)
	}

	// Verify description_gen
	if product.DescriptionGenStatus != "pending" {
		t.Errorf("description_gen_status = %q, want pending", product.DescriptionGenStatus)
	}

	// Verify content_gen on lesson
	m1 := product.CourseModules[0]
	if m1.Lessons[1].ContentGenStatus != "pending" {
		t.Errorf("lesson 2 content_gen_status = %q, want pending", m1.Lessons[1].ContentGenStatus)
	}

	// Verify lesson properties
	if !m1.Lessons[0].IsFree {
		t.Error("lesson 1 should be is_free")
	}
	if m1.Lessons[0].Duration != "00:10:00" {
		t.Errorf("lesson 1 duration = %q", m1.Lessons[0].Duration)
	}

	m2 := product.CourseModules[1]
	if m2.Lessons[0].DripDays != 7 {
		t.Errorf("lesson drip_days = %d, want 7", m2.Lessons[0].DripDays)
	}
	if !m2.Lessons[1].IsDraft {
		t.Error("lesson should be is_draft")
	}

	// Verify quiz linkage
	if m2.QuizSlug == "" {
		t.Error("module 2 should have quiz_slug")
	}
	m3 := product.CourseModules[2]
	if m3.QuizSlug == "" {
		t.Error("module 3 should have quiz_slug")
	}

	// Verify quiz content
	quiz1 := result.Quizzes[0]
	if quiz1.PassThreshold != 75 {
		t.Errorf("quiz 1 pass_threshold = %d, want 75", quiz1.PassThreshold)
	}
	if quiz1.MaxAttempts != 3 {
		t.Errorf("quiz 1 max_attempts = %d, want 3", quiz1.MaxAttempts)
	}
	if len(quiz1.Questions) != 2 {
		t.Errorf("quiz 1 questions = %d, want 2", len(quiz1.Questions))
	}

	quiz2 := result.Quizzes[1]
	if quiz2.PassThreshold != 80 {
		t.Errorf("quiz 2 pass_threshold = %d, want 80", quiz2.PassThreshold)
	}

	t.Logf("Full LMS fixture compiled successfully:")
	t.Logf("  Products: %d", len(result.Products))
	t.Logf("  Offers: %d", len(result.Offers))
	t.Logf("  Stories: %d", len(result.Stories))
	t.Logf("  Quizzes: %d", len(result.Quizzes))
	t.Logf("  CourseModules: %d", len(product.CourseModules))
	t.Logf("  TotalLessons: %d", product.TotalLessons)
	t.Logf("  Badges: %d", len(result.Badges))
}
