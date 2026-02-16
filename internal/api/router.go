package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aetherdev/aetherdev/internal/agent"
	"github.com/aetherdev/aetherdev/internal/bestpractices"
	"github.com/aetherdev/aetherdev/internal/config"
	"github.com/aetherdev/aetherdev/internal/git"
	"github.com/aetherdev/aetherdev/internal/middleware"
	"github.com/aetherdev/aetherdev/internal/models"
	"github.com/aetherdev/aetherdev/internal/oidc"
)

// Handler holds all dependencies for request handling.
type Handler struct {
	cfg          *config.Config
	store        *models.Store
	gitEngine    *git.Engine
	orchestrator *agent.Orchestrator
	bpEngine     *bestpractices.Engine
	oidcClient   *oidc.Client // nil when OIDC is disabled
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

	// OIDC setup (optional — when not configured, falls back to default admin)
	var oidcClient *oidc.Client
	if cfg.Auth.OIDCEnabled && cfg.Auth.IssuerURL != "" {
		oidcClient, err = oidc.NewClient(
			cfg.Auth.IssuerURL,
			cfg.Auth.ClientID,
			cfg.Auth.ClientSecret,
			cfg.OIDCRedirectURI(),
			cfg.OIDCScopes(),
		)
		if err != nil {
			log.Printf("Warning: OIDC discovery failed (auth will fall back to default admin): %v", err)
		} else {
			log.Printf("OIDC enabled: issuer=%s", cfg.Auth.IssuerURL)
		}
	}

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
		oidcClient:   oidcClient,
		templates:    tmpl,
	}

	// Build the session auth middleware with store + OIDC awareness
	sessionAuth := middleware.NewSessionAuth(store, cfg.Auth.OIDCEnabled && oidcClient != nil)

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

	// Auth routes (no session required)
	r.Get("/auth/login", h.authLogin)
	r.Get("/auth/callback", h.authCallback)
	r.Get("/auth/logout", h.authLogout)
	r.Get("/auth/login-page", h.serveSPA) // SPA renders the login page

	// Current user API (needs session but returns 401 instead of redirect)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(sessionAuth)
		r.Get("/me", h.apiAuthMe)
	})

	// Pages (HTML) — serve SPA index.html as fallback for all routes
	r.Group(func(r chi.Router) {
		r.Use(sessionAuth)
		r.Get("/", h.serveSPA)
		r.Get("/new", h.serveSPA)
		r.Get("/{owner}/{repo}", h.serveSPA)
		r.Get("/{owner}/{repo}/tree/{ref}/*", h.serveSPA)
		r.Get("/{owner}/{repo}/blob/{ref}/*", h.serveSPA)
		r.Get("/{owner}/{repo}/agents", h.serveSPA)
		r.Get("/{owner}/{repo}/mindmaps", h.serveSPA)
		r.Get("/{owner}/{repo}/docs", h.serveSPA)
		r.Get("/{owner}/{repo}/practices", h.serveSPA)
		r.Get("/{owner}/{repo}/workflows", h.serveSPA)
	})

	// API (JSON)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(sessionAuth)

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

		// Workflows — plain-English file stored in the repo at .aetherdev/workflows.md
		r.Get("/repos/{owner}/{repo}/workflows/file", h.apiGetWorkflowFile)
		r.Put("/repos/{owner}/{repo}/workflows/file", h.apiSaveWorkflowFile)

		// Workflow runs — observability for agent-executed workflow steps
		r.Get("/repos/{owner}/{repo}/workflows/runs", h.apiListWorkflowRuns)
		r.Post("/repos/{owner}/{repo}/workflows/runs", h.apiCreateWorkflowRun)
		r.Get("/repos/{owner}/{repo}/workflows/runs/{runID}", h.apiGetWorkflowRun)
		r.Put("/repos/{owner}/{repo}/workflows/runs/{runID}", h.apiUpdateWorkflowRun)

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

// --- Auth Handlers ---

const sessionCookieName = "aetherdev_session"

// authLogin redirects the browser to the IdP's authorization endpoint.
func (h *Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	if h.oidcClient == nil {
		// OIDC not configured — just set a session for the default admin
		h.setAdminSession(w)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	state, err := oidc.GenerateState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	// Store state in a short-lived cookie for CSRF verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, h.oidcClient.AuthURL(state), http.StatusFound)
}

// authCallback handles the IdP redirect after the user authenticates.
func (h *Handler) authCallback(w http.ResponseWriter, r *http.Request) {
	if h.oidcClient == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oidc_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}
	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{Name: "oidc_state", Path: "/", MaxAge: -1})

	// Check for error from IdP
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		desc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("IdP error: %s — %s", errMsg, desc), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "no authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for tokens
	tok, err := h.oidcClient.Exchange(code)
	if err != nil {
		log.Printf("OIDC token exchange failed: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Fetch user info
	userInfo, err := h.oidcClient.FetchUserInfo(tok.AccessToken)
	if err != nil {
		log.Printf("OIDC userinfo failed: %v", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	if userInfo.Email == "" {
		http.Error(w, "IdP did not return an email address", http.StatusBadRequest)
		return
	}

	// Provision or update user
	username := userInfo.PreferredUsername
	if username == "" {
		username = strings.Split(userInfo.Email, "@")[0]
	}
	user, err := h.store.UpsertUserByEmail(userInfo.Email, username, userInfo.Name, userInfo.Picture)
	if err != nil {
		log.Printf("User upsert failed: %v", err)
		http.Error(w, "failed to provision user", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID, err := oidc.GenerateSessionID()
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := h.store.CreateSession(sessionID, user.ID, expiresAt); err != nil {
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// authLogout clears the session and optionally redirects to IdP logout.
func (h *Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		h.store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Path:   "/",
		MaxAge: -1,
	})

	if h.oidcClient != nil {
		base := h.cfg.Server.BaseURL
		if base == "" {
			base = fmt.Sprintf("http://localhost:%d", h.cfg.Server.Port)
		}
		http.Redirect(w, r, h.oidcClient.LogoutURL(base), http.StatusFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// setAdminSession creates a session for the default admin user (OIDC disabled).
func (h *Handler) setAdminSession(w http.ResponseWriter) {
	sessionID, err := oidc.GenerateSessionID()
	if err != nil {
		return
	}
	h.store.CreateSession(sessionID, 1, time.Now().Add(24*time.Hour))
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// apiAuthMe returns the current authenticated user.
func (h *Handler) apiAuthMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)
	user, err := h.store.GetUser(userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":         user,
		"oidc_enabled": h.oidcClient != nil,
	})
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

// workflowFilePath is the conventional location for the plain-English workflow file.
const workflowFilePath = ".aetherdev/workflows.md"

// apiGetWorkflowFile reads the workflow file from the repo. If it doesn't exist
// yet, returns a starter template so the user (or agent) knows the format.
func (h *Handler) apiGetWorkflowFile(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	content, err := h.gitEngine.ReadBlob(owner, repoName, "main", workflowFilePath)
	if err != nil {
		// File doesn't exist yet — return empty with exists=false
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"path":    workflowFilePath,
			"content": "",
			"exists":  false,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":    workflowFilePath,
		"content": string(content),
		"exists":  true,
	})
}

// apiSaveWorkflowFile writes the workflow file into the repo as a git commit.
func (h *Handler) apiSaveWorkflowFile(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	var input struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	commitSHA, err := h.gitEngine.WriteFile(
		owner, repoName, "main",
		workflowFilePath, input.Content,
		"Update AetherDev workflows", "AetherDev Agent", "agent@aetherdev.local",
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save workflow file: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"path":       workflowFilePath,
		"commit_sha": commitSHA,
	})
}

// apiListWorkflowRuns returns the run history for a repo's workflows.
func (h *Handler) apiListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
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

	runs, err := h.store.ListWorkflowRuns(repo.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if runs == nil {
		runs = []models.WorkflowRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

// apiCreateWorkflowRun records a new workflow execution.
func (h *Handler) apiCreateWorkflowRun(w http.ResponseWriter, r *http.Request) {
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

	var input struct {
		TriggeredBy string              `json:"triggered_by"`
		Section     string              `json:"section"`
		Status      string              `json:"status"`
		StepResults []models.StepResult `json:"step_results"`
		Summary     string              `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now()
	run := &models.WorkflowRun{
		RepoID:      repo.ID,
		UserID:      user.ID,
		TriggerBy:   input.TriggeredBy,
		Section:     input.Section,
		Status:      input.Status,
		StepResults: input.StepResults,
		Summary:     input.Summary,
		StartedAt:   &now,
	}
	if run.Status == "" {
		run.Status = "pending"
	}
	if err := h.store.CreateWorkflowRun(run); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// apiGetWorkflowRun returns a single workflow run with step-level detail.
func (h *Handler) apiGetWorkflowRun(w http.ResponseWriter, r *http.Request) {
	runIDStr := chi.URLParam(r, "runID")
	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run ID"})
		return
	}

	run, err := h.store.GetWorkflowRun(runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow run not found"})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// apiUpdateWorkflowRun updates a workflow run (status, step results, summary).
func (h *Handler) apiUpdateWorkflowRun(w http.ResponseWriter, r *http.Request) {
	runIDStr := chi.URLParam(r, "runID")
	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run ID"})
		return
	}

	run, err := h.store.GetWorkflowRun(runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow run not found"})
		return
	}

	var input struct {
		Status      string              `json:"status"`
		StepResults []models.StepResult `json:"step_results"`
		Summary     string              `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if input.Status != "" {
		run.Status = input.Status
	}
	if input.StepResults != nil {
		run.StepResults = input.StepResults
	}
	if input.Summary != "" {
		run.Summary = input.Summary
	}
	if run.Status == "completed" || run.Status == "failed" {
		now := time.Now()
		run.CompletedAt = &now
	}

	if err := h.store.UpdateWorkflowRun(run); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
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
