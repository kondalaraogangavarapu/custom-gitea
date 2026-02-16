package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store provides data access to the AetherDev database.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			full_name TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			is_admin BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_id INTEGER NOT NULL REFERENCES users(id),
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			is_private BOOLEAN DEFAULT 0,
			default_branch TEXT DEFAULT 'main',
			language TEXT DEFAULT '',
			stars INTEGER DEFAULT 0,
			forks INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(owner_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS agent_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL REFERENCES repositories(id),
			user_id INTEGER NOT NULL REFERENCES users(id),
			type TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			prompt TEXT DEFAULT '',
			result TEXT DEFAULT '',
			provider TEXT DEFAULT 'claude',
			model TEXT DEFAULT '',
			tokens_used INTEGER DEFAULT 0,
			branch_name TEXT DEFAULT '',
			commit_sha TEXT DEFAULT '',
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL REFERENCES repositories(id),
			task_id INTEGER REFERENCES agent_tasks(id),
			title TEXT NOT NULL,
			type TEXT NOT NULL,
			content TEXT DEFAULT '',
			path TEXT DEFAULT '',
			version INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS mind_maps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL REFERENCES repositories(id),
			title TEXT NOT NULL,
			root_node_json TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS workflow_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL REFERENCES repositories(id),
			user_id INTEGER NOT NULL REFERENCES users(id),
			triggered_by TEXT DEFAULT '',
			section TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			step_results_json TEXT DEFAULT '[]',
			summary TEXT DEFAULT '',
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Seed a default admin user for v1
		`INSERT OR IGNORE INTO users (id, username, email, full_name, is_admin)
		 VALUES (1, 'admin', 'admin@aetherdev.local', 'Administrator', 1)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}
	return nil
}

// --- User operations ---

func (s *Store) GetUser(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, email, full_name, avatar_url, is_admin, created_at, updated_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, email, full_name, avatar_url, is_admin, created_at, updated_at FROM users WHERE username=?`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// --- Repository operations ---

func (s *Store) CreateRepository(r *Repository) error {
	res, err := s.db.Exec(
		`INSERT INTO repositories (owner_id, name, description, is_private, default_branch) VALUES (?,?,?,?,?)`,
		r.OwnerID, r.Name, r.Description, r.IsPrivate, r.DefaultBr,
	)
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	return nil
}

func (s *Store) ListRepositories(ownerID int64) ([]Repository, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_id, name, description, is_private, default_branch, language, stars, forks, created_at, updated_at
		 FROM repositories WHERE owner_id=? ORDER BY updated_at DESC`, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.Name, &r.Description, &r.IsPrivate, &r.DefaultBr, &r.Language, &r.Stars, &r.Forks, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, nil
}

func (s *Store) GetRepository(ownerID int64, name string) (*Repository, error) {
	r := &Repository{}
	err := s.db.QueryRow(
		`SELECT id, owner_id, name, description, is_private, default_branch, language, stars, forks, created_at, updated_at
		 FROM repositories WHERE owner_id=? AND name=?`, ownerID, name,
	).Scan(&r.ID, &r.OwnerID, &r.Name, &r.Description, &r.IsPrivate, &r.DefaultBr, &r.Language, &r.Stars, &r.Forks, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// --- Agent task operations ---

func (s *Store) CreateAgentTask(t *AgentTask) error {
	res, err := s.db.Exec(
		`INSERT INTO agent_tasks (repo_id, user_id, type, status, prompt, provider, model, branch_name)
		 VALUES (?,?,?,?,?,?,?,?)`,
		t.RepoID, t.UserID, t.Type, "pending", t.Prompt, t.Provider, t.Model, t.BranchName,
	)
	if err != nil {
		return err
	}
	t.ID, _ = res.LastInsertId()
	t.CreatedAt = time.Now()
	return nil
}

func (s *Store) UpdateAgentTask(t *AgentTask) error {
	_, err := s.db.Exec(
		`UPDATE agent_tasks SET status=?, result=?, tokens_used=?, commit_sha=?, started_at=?, completed_at=? WHERE id=?`,
		t.Status, t.Result, t.TokensUsed, t.CommitSHA, t.StartedAt, t.CompletedAt, t.ID,
	)
	return err
}

func (s *Store) ListAgentTasks(repoID int64) ([]AgentTask, error) {
	rows, err := s.db.Query(
		`SELECT id, repo_id, user_id, type, status, prompt, result, provider, model, tokens_used, branch_name, commit_sha, started_at, completed_at, created_at
		 FROM agent_tasks WHERE repo_id=? ORDER BY created_at DESC LIMIT 50`, repoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []AgentTask
	for rows.Next() {
		var t AgentTask
		if err := rows.Scan(&t.ID, &t.RepoID, &t.UserID, &t.Type, &t.Status, &t.Prompt, &t.Result, &t.Provider, &t.Model, &t.TokensUsed, &t.BranchName, &t.CommitSHA, &t.StartedAt, &t.CompletedAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// --- Document operations ---

func (s *Store) CreateDocument(d *Document) error {
	res, err := s.db.Exec(
		`INSERT INTO documents (repo_id, task_id, title, type, content, path) VALUES (?,?,?,?,?,?)`,
		d.RepoID, d.TaskID, d.Title, d.Type, d.Content, d.Path,
	)
	if err != nil {
		return err
	}
	d.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) ListDocuments(repoID int64) ([]Document, error) {
	rows, err := s.db.Query(
		`SELECT id, repo_id, task_id, title, type, path, version, created_at, updated_at
		 FROM documents WHERE repo_id=? ORDER BY updated_at DESC`, repoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []Document
	for _, _ = range []int{} {
		// placeholder
		break
	}
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.RepoID, &d.TaskID, &d.Title, &d.Type, &d.Path, &d.Version, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

// --- Workflow run operations ---

func (s *Store) CreateWorkflowRun(r *WorkflowRun) error {
	resultsJSON, err := json.Marshal(r.StepResults)
	if err != nil {
		return fmt.Errorf("marshal step_results: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO workflow_runs (repo_id, user_id, triggered_by, section, status, step_results_json, summary, started_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		r.RepoID, r.UserID, r.TriggerBy, r.Section, r.Status, string(resultsJSON), r.Summary, r.StartedAt,
	)
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	r.CreatedAt = time.Now()
	return nil
}

func (s *Store) UpdateWorkflowRun(r *WorkflowRun) error {
	resultsJSON, err := json.Marshal(r.StepResults)
	if err != nil {
		return fmt.Errorf("marshal step_results: %w", err)
	}
	_, err = s.db.Exec(
		`UPDATE workflow_runs SET status=?, step_results_json=?, summary=?, started_at=?, completed_at=? WHERE id=?`,
		r.Status, string(resultsJSON), r.Summary, r.StartedAt, r.CompletedAt, r.ID,
	)
	return err
}

func (s *Store) GetWorkflowRun(id int64) (*WorkflowRun, error) {
	r := &WorkflowRun{}
	var resultsJSON string
	err := s.db.QueryRow(
		`SELECT id, repo_id, user_id, triggered_by, section, status, step_results_json, summary, started_at, completed_at, created_at
		 FROM workflow_runs WHERE id=?`, id,
	).Scan(&r.ID, &r.RepoID, &r.UserID, &r.TriggerBy, &r.Section, &r.Status, &resultsJSON, &r.Summary, &r.StartedAt, &r.CompletedAt, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(resultsJSON), &r.StepResults); err != nil {
		return nil, fmt.Errorf("unmarshal step_results: %w", err)
	}
	return r, nil
}

func (s *Store) ListWorkflowRuns(repoID int64) ([]WorkflowRun, error) {
	rows, err := s.db.Query(
		`SELECT id, repo_id, user_id, triggered_by, section, status, step_results_json, summary, started_at, completed_at, created_at
		 FROM workflow_runs WHERE repo_id=? ORDER BY created_at DESC LIMIT 50`, repoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []WorkflowRun
	for rows.Next() {
		var r WorkflowRun
		var resultsJSON string
		if err := rows.Scan(&r.ID, &r.RepoID, &r.UserID, &r.TriggerBy, &r.Section, &r.Status, &resultsJSON, &r.Summary, &r.StartedAt, &r.CompletedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(resultsJSON), &r.StepResults); err != nil {
			return nil, fmt.Errorf("unmarshal step_results: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, nil
}

// --- Session operations ---

func (s *Store) CreateSession(id string, userID int64, expiresAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?,?,?)`,
		id, userID, expiresAt,
	)
	return err
}

func (s *Store) GetSession(id string) (int64, error) {
	var userID int64
	var expiresAt time.Time
	err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE id=?`, id,
	).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, err
	}
	if time.Now().After(expiresAt) {
		s.db.Exec(`DELETE FROM sessions WHERE id=?`, id)
		return 0, fmt.Errorf("session expired")
	}
	return userID, nil
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return err
}

// UpsertUserByEmail creates or updates a user matched by email.
// Used during OIDC login to provision users on first sign-in.
func (s *Store) UpsertUserByEmail(email, username, fullName, avatarURL string) (*User, error) {
	// Try update first
	res, err := s.db.Exec(
		`UPDATE users SET username=?, full_name=?, avatar_url=?, updated_at=CURRENT_TIMESTAMP WHERE email=?`,
		username, fullName, avatarURL, email,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return s.GetUserByEmail(email)
	}
	// Insert new user
	_, err = s.db.Exec(
		`INSERT INTO users (username, email, full_name, avatar_url, is_admin) VALUES (?,?,?,?,0)`,
		username, email, fullName, avatarURL,
	)
	if err != nil {
		return nil, err
	}
	return s.GetUserByEmail(email)
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, email, full_name, avatar_url, is_admin, created_at, updated_at FROM users WHERE email=?`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}
