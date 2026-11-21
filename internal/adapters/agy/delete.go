package agy

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
	_ "modernc.org/sqlite"
)

func (Adapter) DeleteScope() string { return "session and all subagent conversations" }

// commitTx and sqliteBusyTimeout are overridden in tests.
var (
	commitTx          = func(tx *sql.Tx) error { return tx.Commit() }
	sqliteBusyTimeout = 5000
)

// Delete removes an Antigravity conversation and all of its subagent
// descendants. It refuses a conversation that is open in a running agy.
func (Adapter) Delete(session resume.Session) error {
	root, err := dataRoot()
	if err != nil {
		return err
	}
	return deleteSessionInRoot(root, session)
}

func validID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	if strings.ContainsAny(id, `/\`) {
		return false
	}
	return filepath.Base(id) == id
}

// deleteSessionInRoot deletes a conversation and its subagent descendants:
//  1. Acquire stable presence flocks (creating missing lockfiles; never unlinking them).
//  2. Start an EXCLUSIVE SQLite transaction and execute all DELETEs before
//     touching files, so schema drift, read-only DBs, concurrent writers, and
//     rollback-journal readers fail up front instead of blocking COMMIT.
//  3. Validate file-only artifact paths do not contain directories, then remove
//     local artifacts (brain trees first, then files); roll back SQLite on error.
//  4. Commit the SQLite transaction.
func deleteSessionInRoot(root string, session resume.Session) error {
	id := session.ID
	if !validID(id) {
		return fmt.Errorf("agy: invalid conversation id %q", id)
	}
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var unlocks []func()
	defer func() {
		for i := len(unlocks) - 1; i >= 0; i-- {
			unlocks[i]()
		}
	}()
	locked := map[string]bool{}
	acquireLock := func(cid string) error {
		if !validID(cid) || locked[cid] {
			return nil
		}
		unlock, err := lockPresence(root, cid)
		if err != nil {
			return err
		}
		locked[cid] = true
		unlocks = append(unlocks, unlock)
		return nil
	}

	if err := acquireLock(id); err != nil {
		return err
	}

	ids := []string{id}
	seen := map[string]bool{id: true}
	var (
		db *sql.DB
		tx *sql.Tx
	)
	dbPath := filepath.Join(root, summariesDB)
	if _, err := os.Stat(dbPath); err == nil {
		dsn := fmt.Sprintf("_pragma=busy_timeout(%d)&_txlock=exclusive", sqliteBusyTimeout)
		db, err = sql.Open("sqlite", common.SQLiteURI(dbPath, dsn))
		if err != nil {
			return err
		}
		defer db.Close()

		tx, err = db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		descendants, err := descendantIDs(tx, id)
		if err != nil {
			return err
		}
		for _, child := range descendants {
			if validID(child) && !seen[child] {
				seen[child] = true
				ids = append(ids, child)
			}
		}
		for _, cid := range ids[1:] {
			if err := acquireLock(cid); err != nil {
				return err
			}
		}

		if err := deleteRowsTx(tx, ids); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	targets, err := collectArtifacts(root, ids)
	if err != nil {
		return err
	}
	if err := removeArtifacts(targets); err != nil {
		return err
	}

	if tx == nil {
		return nil
	}
	return commitTx(tx)
}

// lockPresence opens or creates the shared presence lockfile and takes the
// exclusive flock that a running agy holds on an open conversation. Existing
// lockfiles are opened read-only so a 0444 lockfile does not block deletion,
// and lockfiles are kept in place so their inodes stay stable across lockers.
func lockPresence(root, id string) (func(), error) {
	presenceDir := filepath.Join(root, "presence")
	if err := os.MkdirAll(presenceDir, 0o755); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(presenceDir, id+".lock")
	for range 8 {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDONLY, 0o600)
		if err != nil {
			return nil, err
		}
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			file.Close()
			if errors.Is(err, syscall.EWOULDBLOCK) {
				return nil, errors.New("agy: conversation is open in a running agy; close it first")
			}
			return nil, err
		}
		fdInfo, fdErr := file.Stat()
		pathInfo, pathErr := os.Stat(lockPath)
		if fdErr == nil && pathErr == nil && os.SameFile(fdInfo, pathInfo) {
			return func() { file.Close() }, nil
		}
		file.Close()
	}
	return nil, fmt.Errorf("agy: presence lock %q changed concurrently", lockPath)
}

type artifactTarget struct {
	path  string
	isDir bool
}

// collectArtifacts lists existing artifact paths for ids and rejects
// directories at file-only paths before any files are removed. Descendants are
// ordered before the root parent (and brain trees before conversation DBs) so a
// child permission error leaves the parent untouched and retryable.
func collectArtifacts(root string, ids []string) ([]artifactTarget, error) {
	type candidate struct {
		path     string
		allowDir bool
	}
	var targets []artifactTarget
	for i := len(ids) - 1; i >= 0; i-- {
		cid := ids[i]
		for _, cand := range []candidate{
			{filepath.Join(root, "brain", cid), true},
			{filepath.Join(root, "annotations", cid+".pbtxt"), false},
			{filepath.Join(root, "conversations", cid+".db"), false},
			{filepath.Join(root, "conversations", cid+".db-wal"), false},
			{filepath.Join(root, "conversations", cid+".db-shm"), false},
		} {
			info, err := os.Lstat(cand.path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if info.IsDir() && !cand.allowDir {
				return nil, fmt.Errorf("agy: refusing to delete directory at file path %q", cand.path)
			}
			targets = append(targets, artifactTarget{path: cand.path, isDir: info.IsDir()})
		}
	}
	return targets, nil
}

func removeArtifacts(targets []artifactTarget) error {
	for _, target := range targets {
		var err error
		if target.isDir {
			err = os.RemoveAll(target.path)
		} else {
			err = os.Remove(target.path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// descendantIDs returns every subagent conversation below id, at any depth.
func descendantIDs(tx *sql.Tx, id string) ([]string, error) {
	// An empty id must never recurse: top-level conversations all have an
	// empty parent and would be swept in as its children.
	rows, err := tx.Query(`
WITH RECURSIVE tree(id) AS (
	SELECT conversation_id FROM conversation_summaries WHERE parent_conversation_id = ?
	UNION
	SELECT s.conversation_id FROM conversation_summaries s
	JOIN tree ON s.parent_conversation_id = tree.id
	WHERE tree.id != ''
)
SELECT id FROM tree`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var child sql.NullString
		if err := rows.Scan(&child); err != nil {
			return nil, err
		}
		ids = append(ids, child.String)
	}
	return ids, rows.Err()
}

// deleteRowsTx removes the conversations from the summaries DB inside tx.
// The side tables may be absent, but when they exist a failure there fails the
// transaction so no rows or files are lost.
func deleteRowsTx(tx *sql.Tx, ids []string) error {
	stmts := []string{`DELETE FROM conversation_summaries WHERE conversation_id = ?1`}
	for _, side := range []struct{ table, stmt string }{
		{"conversations_fts", `DELETE FROM conversations_fts WHERE conversation_id = ?1`},
		{"conversation_parents", `DELETE FROM conversation_parents WHERE conversation_id = ?1 OR parent_id = ?1`},
	} {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, side.table).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			stmts = append(stmts, side.stmt)
		}
	}

	for _, id := range ids {
		for _, stmt := range stmts {
			if _, err := tx.Exec(stmt, id); err != nil {
				return err
			}
		}
	}
	return nil
}
