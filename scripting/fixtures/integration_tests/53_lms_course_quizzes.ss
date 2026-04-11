// LMS Course with Quizzes and AI-Generated Content
// Tests the full LMS scripting pipeline:
// - Course product with instructor and description_gen
// - Modules with ordered lessons
// - Lesson features: is_free, drip_days, duration, content_gen
// - LMS quizzes with multiple_choice and short_answer questions
// - Badge grants for enrollment
// - Offer pricing linked to course product

product "Kubernetes Mastery" {
	type course
	description "Master Kubernetes from development to production"
	instructor "DevOps Expert"

	description_gen {
		instruction "Create a comprehensive course description for a Kubernetes course"
		references ["https://kubernetes.io/docs"]
	}

	module "Foundations" {
		lesson "What is Kubernetes?" {
			video_url "https://cdn.example.com/k8s-intro.mp4"
			content "<h2>Introduction</h2><p>Kubernetes is an open-source container orchestration platform.</p>"
			is_free true
			duration "00:15:00"
		}
		lesson "Architecture Overview" {
			video_url "https://cdn.example.com/k8s-arch.mp4"
			description_gen {
				instruction "Explain Kubernetes architecture: control plane, nodes, pods, services"
				references ["https://kubernetes.io/docs/concepts/architecture/"]
				theme "technical"
			}
			drip_days 3
		}
	}

	module "Core Concepts" {
		lesson "Pods and Deployments" {
			video_url "https://cdn.example.com/k8s-pods.mp4"
			content "<h2>Pods</h2><p>The smallest deployable unit in Kubernetes.</p>"
			drip_days 7
		}
		lesson "Services and Networking" {
			video_url "https://cdn.example.com/k8s-services.mp4"
			content "<h2>Services</h2><p>Abstract way to expose applications.</p>"
		}
		lesson "Draft Lesson" {
			video_url "https://cdn.example.com/draft.mp4"
			content "<p>Coming soon.</p>"
			is_draft true
		}

		quiz "Core Concepts Assessment" {
			pass_threshold 75
			max_attempts 3
			question "What is a Pod?" {
				type multiple_choice
				options ["A container", "A group of containers", "A virtual machine", "A namespace"]
				answer 1
			}
			question "What is the purpose of a Service?" {
				type multiple_choice
				options ["Run containers", "Store data", "Expose applications", "Monitor health"]
				answer 2
			}
			question "Explain the difference between a Deployment and a StatefulSet" {
				type short_answer
				answer "Deployments manage stateless applications while StatefulSets manage stateful applications with persistent identities"
			}
		}
	}

	module "Production Readiness" {
		lesson "Helm Charts" {
			video_url "https://cdn.example.com/helm.mp4"
			content "<h2>Helm</h2><p>The package manager for Kubernetes.</p>"
		}
		lesson "Monitoring with Prometheus" {
			media_ref "prometheus-monitoring-video"
			description_gen {
				instruction "Create lesson about Prometheus monitoring in K8s clusters"
				theme "practical"
			}
		}

		quiz "Production Quiz" {
			pass_threshold 80
			question "What is Helm used for?" {
				type multiple_choice
				options ["Package management for K8s", "Container runtime", "Cluster networking", "Secret management"]
				answer 0
			}
		}
	}
}

offer "K8s Mastery All Access" {
	pricing_model one_time
	price 497.00
	currency "usd"
	includes_product "Kubernetes Mastery"
	grants_badge "k8s_enrolled"
}

story "K8s Welcome Flow" {
	start_trigger "k8s_enrolled"

	storyline "Onboarding" {
		enactment "Day 0" {
			scene "Welcome" {
				subject "Welcome to Kubernetes Mastery!"
				body "<h1>Your K8s journey begins now</h1><p>Check out the first lesson.</p>"
				from_email "learn@k8s-academy.com"
				from_name "K8s Mastery"
				reply_to "support@k8s-academy.com"
			}
		}
	}
}
