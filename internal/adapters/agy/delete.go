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
)

// DeleteSession removes an Antigravity conversation and all of its subagent
// descendants: the local artifacts first, then the summary rows. It refuses a
// conversation that is open in a running agy.
func DeleteSession(session resume.Session) error {
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

func deleteSessionInRoot(root string, session resume.Session) error {
	id := session.ID
	if !validID(id) {
		return fmt.Errorf("agy: invalid conversation id %q", id)
	}

	unlock, err := lockPresence(root, id)
	if err != nil {
		return err
	}
	defer unlock()

	ids := []string{id}
	var db *sql.DB
	if dbPath := filepath.Join(root, summariesDB); common.Exists(dbPath) {
		db, err = sql.Open("sqlite", sqliteURI(dbPath, "_pragma=busy_timeout(5000)"))
		if err != nil {
			return err
		}
		defer db.Close()

		descendants, err := descendantIDs(db, id)
		if err != nil {
			return err
		}
		for _, child := range descendants {
			if validID(child) && child != id {
				ids = append(ids, child)
			}
		}
	}

	// Artifacts go first: the rows stay until the files are gone, so a failed
	// delete is still listed and can be retried.
	var errs []error
	for _, cid := range ids {
		for _, path := range []string{
			filepath.Join(root, "conversations", cid+".db"),
			filepath.Join(root, "conversations", cid+".db-wal"),
			filepath.Join(root, "conversations", cid+".db-shm"),
			filepath.Join(root, "annotations", cid+".pbtxt"),
			filepath.Join(root, "presence", cid+".lock"),
		} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err)
			}
		}
		if err := os.RemoveAll(filepath.Join(root, "brain", cid)); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	if db == nil {
		return nil
	}
	return deleteRows(db, ids)
}

// lockPresence takes the flock that a running agy holds on a conversation it
// has open, so a live conversation is never deleted underneath it. The
// returned func releases the lock.
func lockPresence(root, id string) (func(), error) {
	file, err := os.Open(filepath.Join(root, "presence", id+".lock"))
	if errors.Is(err, os.ErrNotExist) {
		return func() {}, nil
	}
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
	return func() { file.Close() }, nil
}

// descendantIDs returns every subagent conversation below id, at any depth.
func descendantIDs(db *sql.DB, id string) ([]string, error) {
	// An empty id must never recurse: top-level conversations all have an
	// empty parent and would be swept in as its children.
	rows, err := db.Query(`
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

// deleteRows removes the conversations from the summaries DB in a single
// transaction. The side tables may be absent, but when they exist a failure
// there rolls the whole delete back.
func deleteRows(db *sql.DB, ids []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	return tx.Commit()
}
