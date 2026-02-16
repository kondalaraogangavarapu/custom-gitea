package bestpractices

import (
	"os"
	"path/filepath"

	"github.com/aetherdev/aetherdev/internal/models"
)

// Engine scans repositories and evaluates adherence to best practices.
type Engine struct{}

// NewEngine creates a new best practices engine.
func NewEngine() *Engine { return &Engine{} }

// Analyze runs all best practice checks against a repository on disk.
func (e *Engine) Analyze(repoPath string) (*models.BestPracticeReport, error) {
	report := &models.BestPracticeReport{
		Categories: []models.PracticeCategory{
			e.checkGitOps(repoPath),
			e.checkSDLC(repoPath),
			e.checkCloud(repoPath),
			e.checkSecurity(repoPath),
		},
	}

	total := 0
	for _, cat := range report.Categories {
		total += cat.Score
	}
	report.OverallScore = total / len(report.Categories)
	return report, nil
}

func (e *Engine) checkGitOps(repoPath string) models.PracticeCategory {
	items := []models.PracticeItem{
		e.fileExists(repoPath, ".gitignore", "gitignore_present", "Repository has .gitignore", "Add a .gitignore file to exclude build artifacts and secrets"),
		e.fileExists(repoPath, ".github/PULL_REQUEST_TEMPLATE.md", "pr_template", "PR template exists", "Add a PR template to standardize code reviews"),
		e.fileExists(repoPath, ".github/CODEOWNERS", "codeowners", "CODEOWNERS file exists", "Add CODEOWNERS to enforce review requirements"),
		e.fileExists(repoPath, ".pre-commit-config.yaml", "pre_commit_hooks", "Pre-commit hooks configured", "Add pre-commit hooks for linting and formatting"),
		e.dirExists(repoPath, ".github/workflows", "ci_workflows", "CI/CD workflows present", "Add GitHub Actions or equivalent CI/CD pipelines"),
		e.fileExists(repoPath, "CHANGELOG.md", "changelog", "Changelog maintained", "Add a CHANGELOG.md to track version changes"),
		e.fileExists(repoPath, ".editorconfig", "editorconfig", "EditorConfig present", "Add .editorconfig for consistent coding styles"),
	}
	return models.PracticeCategory{
		Name:  "gitops",
		Score: e.calcScore(items),
		Items: items,
	}
}

func (e *Engine) checkSDLC(repoPath string) models.PracticeCategory {
	items := []models.PracticeItem{
		e.anyFileExists(repoPath, []string{"*_test.go", "**/*_test.py", "**/*.test.js", "**/*.test.ts", "**/*.spec.ts"}, "tests_present", "Tests exist in repository", "Add unit tests to ensure code quality"),
		e.fileExists(repoPath, "Makefile", "makefile", "Makefile or build script present", "Add a Makefile or build script for reproducible builds"),
		e.anyFileExists(repoPath, []string{".eslintrc*", ".golangci.yml", "pyproject.toml", ".rubocop.yml"}, "linter_config", "Linter configuration present", "Configure a linter for consistent code quality"),
		e.fileExists(repoPath, "Dockerfile", "dockerfile", "Dockerfile present", "Add a Dockerfile for containerized deployments"),
		e.anyFileExists(repoPath, []string{"docker-compose.yml", "docker-compose.yaml"}, "compose", "Docker Compose present", "Add docker-compose for local development environment"),
		e.fileExists(repoPath, "LICENSE", "license", "License file present", "Add a LICENSE file to define usage terms"),
		e.dirExists(repoPath, "docs", "docs_dir", "Documentation directory exists", "Create a docs/ directory for project documentation"),
	}
	return models.PracticeCategory{
		Name:  "sdlc",
		Score: e.calcScore(items),
		Items: items,
	}
}

func (e *Engine) checkCloud(repoPath string) models.PracticeCategory {
	items := []models.PracticeItem{
		e.anyFileExists(repoPath, []string{"terraform/**/*.tf", "*.tf", "pulumi/*", "cdk.json"}, "iac_present", "Infrastructure as Code present", "Define infrastructure with Terraform, Pulumi, or CDK"),
		e.anyFileExists(repoPath, []string{"k8s/**/*.yaml", "kubernetes/**/*.yaml", "helm/**", "kustomization.yaml"}, "k8s_manifests", "Kubernetes manifests present", "Add K8s manifests for container orchestration"),
		e.fileExists(repoPath, ".env.example", "env_example", "Environment template present", "Add .env.example to document required environment variables"),
		e.anyFileExists(repoPath, []string{"monitoring/**", "prometheus.yml", "grafana/**"}, "monitoring", "Monitoring configuration present", "Add monitoring setup (Prometheus, Grafana, etc.)"),
		e.anyFileExists(repoPath, []string{"helm/**", "charts/**"}, "helm_charts", "Helm charts present", "Add Helm charts for Kubernetes deployments"),
	}
	return models.PracticeCategory{
		Name:  "cloud",
		Score: e.calcScore(items),
		Items: items,
	}
}

func (e *Engine) checkSecurity(repoPath string) models.PracticeCategory {
	items := []models.PracticeItem{
		e.fileExists(repoPath, "SECURITY.md", "security_policy", "Security policy present", "Add SECURITY.md with vulnerability reporting guidelines"),
		e.fileExists(repoPath, ".github/dependabot.yml", "dependabot", "Dependabot configured", "Enable Dependabot for automated dependency updates"),
		e.noSecretsInCode(repoPath),
		e.anyFileExists(repoPath, []string{".trivyignore", ".snyk", ".grype.yaml"}, "vuln_scanning", "Vulnerability scanner configured", "Add container/dependency vulnerability scanning"),
		e.anyFileExists(repoPath, []string{".github/workflows/codeql*.yml", ".github/workflows/security*.yml"}, "sast", "SAST pipeline configured", "Add static application security testing to CI/CD"),
	}
	return models.PracticeCategory{
		Name:  "security",
		Score: e.calcScore(items),
		Items: items,
	}
}

// --- Helper functions ---

func (e *Engine) fileExists(repoPath, file, rule, desc, suggestion string) models.PracticeItem {
	path := filepath.Join(repoPath, file)
	if _, err := os.Stat(path); err == nil {
		return models.PracticeItem{Rule: rule, Description: desc, Status: "pass", Severity: "info", Suggestion: "", AutoFixable: true}
	}
	return models.PracticeItem{Rule: rule, Description: desc, Status: "fail", Severity: "medium", Suggestion: suggestion, AutoFixable: true}
}

func (e *Engine) dirExists(repoPath, dir, rule, desc, suggestion string) models.PracticeItem {
	path := filepath.Join(repoPath, dir)
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return models.PracticeItem{Rule: rule, Description: desc, Status: "pass", Severity: "info", Suggestion: "", AutoFixable: true}
	}
	return models.PracticeItem{Rule: rule, Description: desc, Status: "fail", Severity: "medium", Suggestion: suggestion, AutoFixable: true}
}

func (e *Engine) anyFileExists(repoPath string, patterns []string, rule, desc, suggestion string) models.PracticeItem {
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(repoPath, pattern))
		if len(matches) > 0 {
			return models.PracticeItem{Rule: rule, Description: desc, Status: "pass", Severity: "info", Suggestion: "", AutoFixable: true}
		}
	}
	return models.PracticeItem{Rule: rule, Description: desc, Status: "fail", Severity: "medium", Suggestion: suggestion, AutoFixable: true}
}

func (e *Engine) noSecretsInCode(repoPath string) models.PracticeItem {
	// Simplified check - in production, use a proper secrets scanner
	envFile := filepath.Join(repoPath, ".env")
	if _, err := os.Stat(envFile); err == nil {
		// .env exists - check if it's in .gitignore
		gitignore := filepath.Join(repoPath, ".gitignore")
		if data, err := os.ReadFile(gitignore); err == nil {
			for _, line := range filepath.SplitList(string(data)) {
				if line == ".env" || line == ".env*" {
					return models.PracticeItem{
						Rule: "no_secrets", Description: "No secrets committed to repository",
						Status: "pass", Severity: "info", AutoFixable: false,
					}
				}
			}
		}
		return models.PracticeItem{
			Rule: "no_secrets", Description: "No secrets committed to repository",
			Status: "warn", Severity: "high", Suggestion: "Ensure .env is in .gitignore and secrets are not committed", AutoFixable: false,
		}
	}
	return models.PracticeItem{
		Rule: "no_secrets", Description: "No secrets committed to repository",
		Status: "pass", Severity: "info", AutoFixable: false,
	}
}

func (e *Engine) calcScore(items []models.PracticeItem) int {
	if len(items) == 0 {
		return 100
	}
	passed := 0
	for _, item := range items {
		if item.Status == "pass" {
			passed++
		} else if item.Status == "warn" {
			passed++ // partial credit
		}
	}
	return (passed * 100) / len(items)
}
