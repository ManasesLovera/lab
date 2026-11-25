package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	_ "modernc.org/sqlite"
)

type badRequestError struct{ msg string }

func (e badRequestError) Error() string { return e.msg }

func errBadRequest(msg string) error { return badRequestError{msg} }

func isBadRequest(err error) bool {
	var b badRequestError
	return errors.As(err, &b)
}

// Secrets management for the lab. Entries live in shared/data/secrets.db, the
// same SQLite file the `secrets` CLI writes, so both tools stay in sync.
// Project .env files are also readable (and editable) from the dashboard.

var secretsMu sync.Mutex

func labDir() string {
	if d := os.Getenv("LAB_DIR"); d != "" {
		return d
	}
	return hostPath("/home/mlovera/lab")
}

func secretsDBPath() string {
	if p := os.Getenv("SECRETS_DB"); p != "" {
		return p
	}
	return filepath.Join(labDir(), "shared", "data", "secrets.db")
}

func historyEnabled() bool {
	switch strings.ToLower(os.Getenv("SECRETS_HISTORY")) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

const secretsSchema = `
CREATE TABLE IF NOT EXISTS entries (
    id          INTEGER PRIMARY KEY,
    name        TEXT UNIQUE NOT NULL,
    type        TEXT,
    url         TEXT,
    username    TEXT,
    password    TEXT,
    description TEXT,
    tags        TEXT,
    meta        TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS user_creation_log (
    id          INTEGER PRIMARY KEY,
    timestamp   DATETIME DEFAULT CURRENT_TIMESTAMP,
    type        TEXT,
    dbname      TEXT,
    username    TEXT,
    permissions TEXT
);
CREATE TABLE IF NOT EXISTS entry_history (
    id          INTEGER PRIMARY KEY,
    entry_id    INTEGER NOT NULL,
    entry_name  TEXT    NOT NULL,
    field       TEXT,
    old_value   TEXT,
    new_value   TEXT,
    change_type TEXT    NOT NULL,
    changed_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_history_entry ON entry_history(entry_id, changed_at);
CREATE INDEX IF NOT EXISTS idx_history_name  ON entry_history(entry_name, changed_at);
`

func openSecrets() (*sql.DB, error) {
	dsn := "file:" + secretsDBPath() + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(secretsSchema); err != nil {
		db.Close()
		return nil, err
	}
	enforceSecretPerms()
	return db, nil
}

func enforceSecretPerms() {
	path := secretsDBPath()
	_ = os.Chmod(filepath.Dir(path), 0o700)
	if _, err := os.Stat(path); err == nil {
		_ = os.Chmod(path, 0o600)
	}
}

type SecretEntry struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	Meta        string `json:"meta"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	HasPassword bool   `json:"has_password"`
}

func ns(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func scanSecret(row interface{ Scan(...any) error }) (SecretEntry, error) {
	var e SecretEntry
	var typ, url, user, pass, desc, tags, meta sql.NullString
	if err := row.Scan(&e.ID, &e.Name, &typ, &url, &user, &pass, &desc, &tags, &meta,
		&e.CreatedAt, &e.UpdatedAt); err != nil {
		return e, err
	}
	e.Type, e.URL, e.Username = ns(typ), ns(url), ns(user)
	e.Password, e.Description = ns(pass), ns(desc)
	e.Tags, e.Meta = ns(tags), ns(meta)
	e.HasPassword = e.Password != ""
	return e, nil
}

const secretCols = "id, name, type, url, username, password, description, tags, meta, created_at, updated_at"

func listSecrets(reveal bool) ([]SecretEntry, error) {
	out := []SecretEntry{}
	db, err := openSecrets()
	if err != nil {
		return out, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT " + secretCols + " FROM entries ORDER BY name")
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		e, err := scanSecret(rows)
		if err != nil {
			return out, err
		}
		if !reveal {
			e.Password = ""
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func getSecret(name string, reveal bool) (SecretEntry, bool, error) {
	db, err := openSecrets()
	if err != nil {
		return SecretEntry{}, false, err
	}
	defer db.Close()
	row := db.QueryRow("SELECT "+secretCols+" FROM entries WHERE name = ?", name)
	e, err := scanSecret(row)
	if err == sql.ErrNoRows {
		return SecretEntry{}, false, nil
	}
	if err != nil {
		return SecretEntry{}, false, err
	}
	if !reveal {
		e.Password = ""
	}
	return e, true, nil
}

type secretMutation struct {
	Name             string  `json:"name"`
	Type             *string `json:"type"`
	URL              *string `json:"url"`
	Username         *string `json:"username"`
	Password         *string `json:"password"`
	ClearPassword    bool    `json:"clear_password"`
	GeneratePassword bool    `json:"generate_password"`
	Description      *string `json:"description"`
	Tags             *string `json:"tags"`
	Meta             *string `json:"meta"`
}

func generatePassword() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, 28)
	n := big.NewInt(int64(len(alphabet)))
	for i := range b {
		if v, err := rand.Int(rand.Reader, n); err == nil {
			b[i] = alphabet[v.Int64()]
		} else {
			b[i] = alphabet[0]
		}
	}
	return string(b)
}

type historyRow struct {
	ID         int64  `json:"id"`
	EntryID    int64  `json:"entry_id"`
	EntryName  string `json:"entry_name"`
	Field      string `json:"field"`
	OldValue   string `json:"old_value"`
	NewValue   string `json:"new_value"`
	ChangeType string `json:"change_type"`
	ChangedAt  string `json:"changed_at"`
}

func recordHistory(tx *sql.Tx, entryID int64, name, changeType string, diffs [][3]string, snapshot string) error {
	if !historyEnabled() {
		return nil
	}
	if diffs != nil {
		for _, d := range diffs {
			if _, err := tx.Exec(
				"INSERT INTO entry_history (entry_id, entry_name, field, old_value, new_value, change_type) VALUES (?, ?, ?, ?, ?, ?)",
				entryID, name, d[0], nullIfEmpty(d[1]), nullIfEmpty(d[2]), changeType,
			); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := tx.Exec(
		"INSERT INTO entry_history (entry_id, entry_name, field, old_value, new_value, change_type) VALUES (?, ?, NULL, ?, NULL, ?)",
		entryID, name, snapshot, changeType,
	)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func upsertSecret(m secretMutation) (SecretEntry, error) {
	secretsMu.Lock()
	defer secretsMu.Unlock()

	db, err := openSecrets()
	if err != nil {
		return SecretEntry{}, err
	}
	defer db.Close()

	if strings.TrimSpace(m.Name) == "" {
		return SecretEntry{}, errBadRequest("name is required")
	}

	existing, err := scanSecret(db.QueryRow("SELECT "+secretCols+" FROM entries WHERE name = ?", m.Name))
	hadExisting := err == nil
	if err != nil && err != sql.ErrNoRows {
		return SecretEntry{}, err
	}

	fields := map[string]string{}
	if hadExisting {
		fields["type"], fields["url"], fields["username"] = existing.Type, existing.URL, existing.Username
		fields["password"], fields["description"] = existing.Password, existing.Description
		fields["tags"], fields["meta"] = existing.Tags, existing.Meta
	}
	apply := func(dst string, v *string) {
		if v != nil {
			fields[dst] = *v
		}
	}
	apply("type", m.Type)
	apply("url", m.URL)
	apply("username", m.Username)
	apply("description", m.Description)
	apply("tags", m.Tags)
	apply("meta", m.Meta)
	switch {
	case m.ClearPassword:
		fields["password"] = ""
	case m.GeneratePassword:
		fields["password"] = generatePassword()
	case m.Password != nil:
		fields["password"] = *m.Password
	}
	if !hadExisting && fields["type"] == "" {
		fields["type"] = "login"
	}

	tx, err := db.Begin()
	if err != nil {
		return SecretEntry{}, err
	}
	defer tx.Rollback()

	if hadExisting {
		_, err = tx.Exec(
			"UPDATE entries SET type=?, url=?, username=?, password=?, description=?, tags=?, meta=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			nullIfEmpty(fields["type"]), nullIfEmpty(fields["url"]), nullIfEmpty(fields["username"]),
			nullIfEmpty(fields["password"]), nullIfEmpty(fields["description"]),
			nullIfEmpty(fields["tags"]), nullIfEmpty(fields["meta"]), existing.ID,
		)
		if err != nil {
			return SecretEntry{}, err
		}
		var diffs [][3]string
		old := map[string]string{
			"type": existing.Type, "url": existing.URL, "username": existing.Username,
			"password": existing.Password, "description": existing.Description,
			"tags": existing.Tags, "meta": existing.Meta,
		}
		for _, f := range []string{"type", "url", "username", "password", "description", "tags", "meta"} {
			if old[f] != fields[f] {
				diffs = append(diffs, [3]string{f, old[f], fields[f]})
			}
		}
		if err := recordHistory(tx, existing.ID, m.Name, "update", diffs, ""); err != nil {
			return SecretEntry{}, err
		}
	} else {
		res, err := tx.Exec(
			"INSERT INTO entries (name, type, url, username, password, description, tags, meta) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			m.Name, nullIfEmpty(fields["type"]), nullIfEmpty(fields["url"]), nullIfEmpty(fields["username"]),
			nullIfEmpty(fields["password"]), nullIfEmpty(fields["description"]),
			nullIfEmpty(fields["tags"]), nullIfEmpty(fields["meta"]),
		)
		if err != nil {
			return SecretEntry{}, err
		}
		id, _ := res.LastInsertId()
		var diffs [][3]string
		for _, f := range []string{"type", "url", "username", "password", "description", "tags", "meta"} {
			if fields[f] != "" {
				diffs = append(diffs, [3]string{f, "", fields[f]})
			}
		}
		if err := recordHistory(tx, id, m.Name, "create", diffs, ""); err != nil {
			return SecretEntry{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return SecretEntry{}, err
	}
	enforceSecretPerms()

	e, _, err := getSecret(m.Name, true)
	return e, err
}

func deleteSecret(name string) (bool, error) {
	secretsMu.Lock()
	defer secretsMu.Unlock()

	db, err := openSecrets()
	if err != nil {
		return false, err
	}
	defer db.Close()

	existing, err := scanSecret(db.QueryRow("SELECT "+secretCols+" FROM entries WHERE name = ?", name))
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	snapshot, _ := json.Marshal(existing)
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM entries WHERE id = ?", existing.ID); err != nil {
		return false, err
	}
	if err := recordHistory(tx, existing.ID, existing.Name, "delete", nil, string(snapshot)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	enforceSecretPerms()
	return true, nil
}

func secretHistory(name string, limit int, reveal bool) ([]historyRow, error) {
	out := []historyRow{}
	db, err := openSecrets()
	if err != nil {
		return out, err
	}
	defer db.Close()

	sqlStr := "SELECT id, entry_id, entry_name, field, old_value, new_value, change_type, changed_at FROM entry_history"
	args := []any{}
	if name != "" {
		sqlStr += " WHERE entry_name = ?"
		args = append(args, name)
	}
	sqlStr += " ORDER BY changed_at DESC, id DESC"
	if limit > 0 {
		sqlStr += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var h historyRow
		var field, oldV, newV sql.NullString
		if err := rows.Scan(&h.ID, &h.EntryID, &h.EntryName, &field, &oldV, &newV, &h.ChangeType, &h.ChangedAt); err != nil {
			return out, err
		}
		h.Field, h.OldValue, h.NewValue = ns(field), ns(oldV), ns(newV)
		if !reveal {
			if h.Field == "" || h.Field == "password" {
				if h.OldValue != "" {
					h.OldValue = "****"
				}
				if h.NewValue != "" {
					h.NewValue = "****"
				}
			}
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ------------------------------------------------------------------ .env files

type EnvVar struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Sensitive bool   `json:"sensitive"`
}

type EnvFile struct {
	Path string   `json:"path"`
	Vars []EnvVar `json:"vars"`
}

var envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func envSensitive(key string) bool {
	k := strings.ToUpper(key)
	for _, needle := range []string{"PASS", "SECRET", "TOKEN", "KEY", "CRED", "AUTH"} {
		if strings.Contains(k, needle) {
			return true
		}
	}
	return false
}

func envFileDirs() []string {
	base := labDir()
	dirs := []string{filepath.Join(base, ".env")}
	for _, sub := range []string{"core", "external", "services"} {
		root := filepath.Join(base, sub)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dirs = append(dirs, filepath.Join(root, e.Name(), ".env"))
		}
	}
	return dirs
}

func parseEnvFile(path, rel string, reveal bool) (EnvFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EnvFile{}, err
	}
	f := EnvFile{Path: rel, Vars: []EnvVar{}}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, val, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !envKeyRe.MatchString(key) {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		sensitive := envSensitive(key)
		if sensitive && !reveal {
			if val != "" {
				val = "****"
			}
		}
		f.Vars = append(f.Vars, EnvVar{Key: key, Value: val, Sensitive: sensitive})
	}
	return f, nil
}

func listEnvFiles(reveal bool) ([]EnvFile, error) {
	base := labDir()
	files := []EnvFile{}
	for _, path := range envFileDirs() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			rel = path
		}
		f, err := parseEnvFile(path, rel, reveal)
		if err != nil {
			continue
		}
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// resolveEnvPath maps a relative path (as returned by listEnvFiles) back to a
// real, allowed .env file. Anything else is rejected to prevent traversal.
func resolveEnvPath(rel string) (string, bool) {
	base := labDir()
	clean := filepath.Clean(rel)
	for _, path := range envFileDirs() {
		got, err := filepath.Rel(base, path)
		if err == nil && filepath.Clean(got) == clean {
			return path, true
		}
	}
	return "", false
}

func quoteEnvValue(v string) string {
	if v == "" {
		return ""
	}
	if strings.ContainsAny(v, " \t#\"'") {
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		return `"` + v + `"`
	}
	return v
}

func saveEnvValue(rel, key, value string) error {
	if !envKeyRe.MatchString(key) {
		return errBadRequest("invalid key")
	}
	if strings.ContainsAny(value, "\n\r") {
		return errBadRequest("value may not contain newlines")
	}
	path, ok := resolveEnvPath(rel)
	if !ok {
		return errBadRequest("unknown env file")
	}

	secretsMu.Lock()
	defer secretsMu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.Contains(trimmed, "=") {
			continue
		}
		k, _, _ := strings.Cut(trimmed, "=")
		if strings.TrimSpace(k) == key {
			lines[i] = key + "=" + quoteEnvValue(value)
			replaced = true
			break
		}
	}
	if !replaced {
		out := strings.TrimRight(string(data), "\n")
		if out != "" {
			out += "\n"
		}
		out += key + "=" + quoteEnvValue(value) + "\n"
		lines = strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	}

	body := strings.Join(lines, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return writeFilePreserving(path, []byte(body))
}

// writeFilePreserving writes atomically and keeps the original mode/owner so
// edits made from the container don't leave root-owned files on the host.
func writeFilePreserving(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".env.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chmod(tmpName, info.Mode().Perm())
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = os.Chown(tmpName, int(st.Uid), int(st.Gid))
	}
	return os.Rename(tmpName, path)
}

// ------------------------------------------------------------------- handlers

func wantsReveal(r *http.Request) bool {
	switch strings.ToLower(r.URL.Query().Get("reveal")) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if isBadRequest(err) {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func handleSecrets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entries, err := listSecrets(wantsReveal(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, entries)
	case http.MethodPost, http.MethodPut:
		var m secretMutation
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&m); err != nil {
			writeErr(w, errBadRequest("invalid JSON: "+err.Error()))
			return
		}
		e, err := upsertSecret(m)
		if err != nil {
			writeErr(w, err)
			return
		}
		e.Password = ""
		writeJSON(w, http.StatusOK, e)
	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		ok, err := deleteSecret(name)
		if err != nil {
			writeErr(w, err)
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "entry not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSecretEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("name")
	e, ok, err := getSecret(name, wantsReveal(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "entry not found"})
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func handleSecretHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			limit = n
		}
	}
	rows, err := secretHistory(r.URL.Query().Get("name"), limit, wantsReveal(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func handleEnv(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		files, err := listEnvFiles(wantsReveal(r))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, files)
	case http.MethodPost:
		var req struct {
			Path  string `json:"path"`
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, errBadRequest("invalid JSON: "+err.Error()))
			return
		}
		if err := saveEnvValue(req.Path, req.Key, req.Value); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
