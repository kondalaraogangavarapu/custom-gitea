package api

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/aetherdev/aetherdev/internal/agent"
	"github.com/aetherdev/aetherdev/internal/bestpractices"
	"github.com/aetherdev/aetherdev/internal/config"
	"github.com/aetherdev/aetherdev/internal/git"
	"github.com/aetherdev/aetherdev/internal/middleware"
	"github.com/aetherdev/aetherdev/internal/models"
)

// Handler holds all dependencies for request handling.
type Handler struct {
	cfg          *config.Config
	store        *models.Store
	gitEngine    *git.Engine
	orchestrator *agent.Orchestrator
	bpEngine     *bestpractices.Engine
	templates    *template.Template
}

// NewRouter creates and configures the HTTP router.
func NewRouter(cfg *config.Config) (http.Handler, error) {
	store, err := models.NewStore(cfg.DBPath())
	if err != nil {
		return nil, err
	}

	gitEngine := git.NewEngine(cfg.RepoRootPath())

	var claudeAgent *agent.ClaudeAgent
	if cfg.Agent.ClaudeAPIKey != "" {
		claudeAgent = agent.NewClaudeAgent(cfg.Agent.ClaudeAPIKey, cfg.Agent.ClaudeModel)
	}

	var bedrockAgent *agent.AWSBedrockAgent
	if cfg.Agent.AWSAgentID != "" {
		bedrockAgent = agent.NewAWSBedrockAgent(cfg.Agent.AWSRegion, cfg.Agent.AWSAgentID, cfg.Agent.AWSAgentAliasID)
	}

	orchestrator := agent.NewOrchestrator(claudeAgent, bedrockAgent, store)
	bpEngine := bestpractices.NewEngine()

	tmpl, err := template.ParseGlob("web/templates/**/*.html")
	if err != nil {
		log.Printf("Warning: could not parse templates: %v", err)
		tmpl = template.New("empty")
	}

	h := &Handler{
		cfg:          cfg,
		store:        store,
		gitEngine:    gitEngine,
		orchestrator: orchestrator,
		bpEngine:     bpEngine,
		templates:    tmpl,
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)

	// Static files (legacy CSS, images)
	staticFS := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", staticFS))

	// SPA assets (Vite build output served from /assets/)
	r.Handle("/assets/*", http.StripPrefix("/", http.FileServer(http.Dir("web/static/dist"))))

	// Pages (HTML) — serve SPA index.html as fallback for all routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.SessionAuth)
		r.Get("/", h.serveSPA)
		r.Get("/new", h.serveSPA)
		r.Get("/{owner}/{repo}", h.serveSPA)
		r.Get("/{owner}/{repo}/tree/{ref}/*", h.serveSPA)
		r.Get("/{owner}/{repo}/blob/{ref}/*", h.serveSPA)
		r.Get("/{owner}/{repo}/agents", h.serveSPA)
		r.Get("/{owner}/{repo}/mindmaps", h.serveSPA)
		r.Get("/{owner}/{repo}/docs", h.serveSPA)
		r.Get("/{owner}/{repo}/practices", h.serveSPA)
		r.Get("/{owner}/{repo}/pipelines", h.serveSPA)
	})

	// API (JSON)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.SessionAuth)

		// Repositories
		r.Get("/repos", h.apiListRepos)
		r.Post("/repos", h.apiCreateRepo)
		r.Get("/repos/{owner}/{repo}", h.apiGetRepo)
		r.Get("/repos/{owner}/{repo}/branches", h.apiListBranches)
		r.Get("/repos/{owner}/{repo}/commits/{ref}", h.apiListCommits)
		r.Get("/repos/{owner}/{repo}/tree/{ref}/*", h.apiListTree)
		r.Get("/repos/{owner}/{repo}/blob/{ref}/*", h.apiReadBlob)

		// Agent tasks
		r.Post("/repos/{owner}/{repo}/agent/tasks", h.apiCreateTask)
		r.Get("/repos/{owner}/{repo}/agent/tasks", h.apiListTasks)

		// Documents
		r.Get("/repos/{owner}/{repo}/documents", h.apiListDocuments)

		// Best practices
		r.Get("/repos/{owner}/{repo}/practices", h.apiAnalyzePractices)

		// System
		r.Get("/health", h.apiHealth)
		r.Get("/version", h.apiVersion)
	})

	return r, nil
}

// --- JSON API Handlers ---

func (h *Handler) apiHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) apiVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"name":    h.cfg.Branding.AppName,
		"version": "0.1.0",
	})
}

func (h *Handler) apiListRepos(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)
	repos, err := h.store.ListRepositories(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if repos == nil {
		repos = []models.Repository{}
	}
	writeJSON(w, http.StatusOK, repos)
}

func (h *Handler) apiCreateRepo(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)
	user, err := h.store.GetUser(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user not found"})
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPrivate   bool   `json:"is_private"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	repo := &models.Repository{
		OwnerID:     userID,
		Name:        input.Name,
		Description: input.Description,
		IsPrivate:   input.IsPrivate,
		DefaultBr:   "main",
	}
	if err := h.store.CreateRepository(repo); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := h.gitEngine.InitRepo(user.Username, input.Name, "main"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to init git repo: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, repo)
}

func (h *Handler) apiGetRepo(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	user, err := h.store.GetUserByUsername(owner)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "owner not found"})
		return
	}
	repo, err := h.store.GetRepository(user.ID, repoName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

func (h *Handler) apiListBranches(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	branches, err := h.gitEngine.ListBranches(owner, repoName)
	if err != nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	writeJSON(w, http.StatusOK, branches)
}

func (h *Handler) apiListCommits(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	ref := chi.URLParam(r, "ref")
	commits, err := h.gitEngine.Log(owner, repoName, ref, 30)
	if err != nil {
		writeJSON(w, http.StatusOK, []git.CommitInfo{})
		return
	}
	writeJSON(w, http.StatusOK, commits)
}

func (h *Handler) apiListTree(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	ref := chi.URLParam(r, "ref")
	path := chi.URLParam(r, "*")
	entries, err := h.gitEngine.ListTree(owner, repoName, ref, path)
	if err != nil {
		writeJSON(w, http.StatusOK, []git.TreeEntry{})
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) apiReadBlob(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	ref := chi.URLParam(r, "ref")
	path := chi.URLParam(r, "*")
	content, err := h.gitEngine.ReadBlob(owner, repoName, ref, path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": string(content), "path": path})
}

func (h *Handler) apiCreateTask(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	user, err := h.store.GetUserByUsername(owner)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "owner not found"})
		return
	}
	repo, err := h.store.GetRepository(user.ID, repoName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}

	var input agent.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	input.RepoID = repo.ID
	input.UserID = user.ID

	task, err := h.orchestrator.ExecuteTask(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) apiListTasks(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	user, err := h.store.GetUserByUsername(owner)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "owner not found"})
		return
	}
	repo, err := h.store.GetRepository(user.ID, repoName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}

	tasks, err := h.store.ListAgentTasks(repo.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []models.AgentTask{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) apiListDocuments(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	user, err := h.store.GetUserByUsername(owner)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "owner not found"})
		return
	}
	repo, err := h.store.GetRepository(user.ID, repoName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}

	docs, err := h.store.ListDocuments(repo.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if docs == nil {
		docs = []models.Document{}
	}
	writeJSON(w, http.StatusOK, docs)
}

func (h *Handler) apiAnalyzePractices(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	repoPath := h.gitEngine.RepoPath(owner, repoName)

	report, err := h.bpEngine.Analyze(repoPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// --- Page Handlers (HTML) ---

type pageData struct {
	Title     string
	AppName   string
	Theme     string
	Accent    string
	User      *models.User
	Repos     []models.Repository
	Repo      *models.Repository
	Owner     string
	RepoName  string
	Ref       string
	Path      string
	TreeItems []git.TreeEntry
	Tasks     []models.AgentTask
	Documents []models.Document
	Report    *models.BestPracticeReport
}

func (h *Handler) newPageData(title string) pageData {
	return pageData{
		Title:   title,
		AppName: h.cfg.Branding.AppName,
		Theme:   h.cfg.Branding.Theme,
		Accent:  h.cfg.Branding.Accent,
	}
}

func (h *Handler) pageDashboard(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)
	user, _ := h.store.GetUser(userID)
	repos, _ := h.store.ListRepositories(userID)

	data := h.newPageData("Dashboard")
	data.User = user
	data.Repos = repos

	h.render(w, "dashboard", data)
}

func (h *Handler) pageNewRepo(w http.ResponseWriter, r *http.Request) {
	data := h.newPageData("New Repository")
	h.render(w, "new_repo", data)
}

func (h *Handler) pageRepo(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	data := h.newPageData(owner + "/" + repoName)
	data.Owner = owner
	data.RepoName = repoName
	data.Ref = "main"

	user, _ := h.store.GetUserByUsername(owner)
	if user != nil {
		repo, _ := h.store.GetRepository(user.ID, repoName)
		data.Repo = repo
	}
	entries, _ := h.gitEngine.ListTree(owner, repoName, "main", "")
	data.TreeItems = entries

	h.render(w, "repo", data)
}

func (h *Handler) pageRepoTree(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	ref := chi.URLParam(r, "ref")
	path := chi.URLParam(r, "*")

	data := h.newPageData(owner + "/" + repoName)
	data.Owner = owner
	data.RepoName = repoName
	data.Ref = ref
	data.Path = path

	entries, _ := h.gitEngine.ListTree(owner, repoName, ref, path)
	data.TreeItems = entries

	h.render(w, "repo_tree", data)
}

func (h *Handler) pageAgents(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	data := h.newPageData("Agents - " + repoName)
	data.Owner = owner
	data.RepoName = repoName

	user, _ := h.store.GetUserByUsername(owner)
	if user != nil {
		repo, _ := h.store.GetRepository(user.ID, repoName)
		if repo != nil {
			data.Tasks, _ = h.store.ListAgentTasks(repo.ID)
		}
	}

	h.render(w, "agents", data)
}

func (h *Handler) pageMindMaps(w http.ResponseWriter, r *http.Request) {
	data := h.newPageData("Mind Maps")
	data.Owner = chi.URLParam(r, "owner")
	data.RepoName = chi.URLParam(r, "repo")
	h.render(w, "mindmaps", data)
}

func (h *Handler) pageDocs(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	data := h.newPageData("Documents - " + repoName)
	data.Owner = owner
	data.RepoName = repoName

	user, _ := h.store.GetUserByUsername(owner)
	if user != nil {
		repo, _ := h.store.GetRepository(user.ID, repoName)
		if repo != nil {
			data.Documents, _ = h.store.ListDocuments(repo.ID)
		}
	}

	h.render(w, "docs", data)
}

func (h *Handler) pagePractices(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	data := h.newPageData("Best Practices - " + repoName)
	data.Owner = owner
	data.RepoName = repoName

	repoPath := h.gitEngine.RepoPath(owner, repoName)
	data.Report, _ = h.bpEngine.Analyze(repoPath)

	h.render(w, "practices", data)
}

func (h *Handler) pagePipelines(w http.ResponseWriter, r *http.Request) {
	data := h.newPageData("Pipelines")
	data.Owner = chi.URLParam(r, "owner")
	data.RepoName = chi.URLParam(r, "repo")
	h.render(w, "pipelines", data)
}

// serveSPA serves the Vite-built SPA index.html for all client-side routes.
func (h *Handler) serveSPA(w http.ResponseWriter, r *http.Request) {
	spaIndex := filepath.Join("web", "static", "dist", "index.html")
	http.ServeFile(w, r, spaIndex)
}

func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error %s: %v", name, err)
		// Fallback: serve the SPA shell
		h.serveSPA(w, nil)
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func getIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
