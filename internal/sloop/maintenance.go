package sloop

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RebuildIndex validates every immutable revision object and reconstructs the
// revision index. It refuses divergent or incomplete v0.1 histories.
func (s *Store) RebuildIndex(ctx context.Context, projectID string) (int, error) {
	objects, err := s.readAllRevisionObjects(projectID)
	if err != nil {
		return 0, err
	}
	bySpec := make(map[string][]Revision)
	allHashes := make(map[string]Revision)
	for _, revision := range objects {
		if _, duplicate := allHashes[revision.Hash]; duplicate {
			return 0, fmt.Errorf("duplicate revision object %s", revision.Hash)
		}
		allHashes[revision.Hash] = revision
		bySpec[revision.SpecificationUUID] = append(bySpec[revision.SpecificationUUID], revision)
	}
	heads := make(map[string]Revision)
	for specUUID, revisions := range bySpec {
		children := make(map[string]int)
		for _, revision := range revisions {
			for _, parent := range revision.ParentRevisionHashes {
				parentObject, ok := allHashes[parent]
				if !ok || parentObject.SpecificationUUID != specUUID {
					return 0, fmt.Errorf("revision %s has missing parent %s", revision.Hash, parent)
				}
				children[parent]++
			}
		}
		var candidates []Revision
		for _, revision := range revisions {
			if children[revision.Hash] == 0 {
				candidates = append(candidates, revision)
			}
		}
		if len(candidates) != 1 {
			return 0, fmt.Errorf("specification %s has %d revision heads; v0.1 cannot resolve divergent history", revisions[0].SpecificationID, len(candidates))
		}
		heads[specUUID] = candidates[0]
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM revision_parents; DELETE FROM revision_index;`); err != nil {
		return 0, fmt.Errorf("clear revision index: %w", err)
	}
	for specUUID, revisions := range bySpec {
		sort.Slice(revisions, func(i, j int) bool {
			if revisions[i].RevisionNumber != revisions[j].RevisionNumber {
				return revisions[i].RevisionNumber < revisions[j].RevisionNumber
			}
			if !revisions[i].CreatedAt.Equal(revisions[j].CreatedAt) {
				return revisions[i].CreatedAt.Before(revisions[j].CreatedAt)
			}
			return revisions[i].Hash < revisions[j].Hash
		})
		for index, revision := range revisions {
			objectPath := s.objectPath(revision.Hash)
			if _, err := tx.ExecContext(ctx, `INSERT INTO revision_index
                (hash,spec_uuid,spec_id,revision_number,status,title,author_name,author_email,author_agent,created_at,object_path)
                VALUES(?,?,?,?,?,?,?,?,?,?,?)`, revision.Hash, revision.SpecificationUUID,
				revision.SpecificationID, index+1, revision.Status, revision.Title, revision.Author.Name,
				revision.Author.Email, revision.Author.Agent, revision.CreatedAt.Format(time.RFC3339Nano), objectPath); err != nil {
				return 0, fmt.Errorf("rebuild revision index: %w", err)
			}
			for _, parent := range revision.ParentRevisionHashes {
				if _, err := tx.ExecContext(ctx, `INSERT INTO revision_parents(revision_hash,parent_hash) VALUES(?,?)`, revision.Hash, parent); err != nil {
					return 0, fmt.Errorf("rebuild revision parents: %w", err)
				}
			}
		}
		head := heads[specUUID]
		var existingDirty int
		err := tx.QueryRowContext(ctx, `SELECT dirty FROM specifications WHERE uuid=?`, specUUID).Scan(&existingDirty)
		if err != nil && err != sql.ErrNoRows {
			return 0, err
		}
		if err == sql.ErrNoRows {
			number, err := specificationNumber(head.SpecificationID)
			if err != nil {
				return 0, err
			}
			parents, _ := json.Marshal(head.Parents)
			if _, err := tx.ExecContext(ctx, `INSERT INTO specifications
                (uuid,id,number,title,status,body,parents_json,author_name,author_email,author_agent,updated_at,head_hash,dirty)
                VALUES(?,?,?,?,?,?,?,?,?,?,?,?,0)`, head.SpecificationUUID, head.SpecificationID, number,
				head.Title, head.Status, head.Content, string(parents), head.Author.Name, head.Author.Email,
				head.Author.Agent, head.CreatedAt.Format(time.RFC3339Nano), head.Hash); err != nil {
				return 0, fmt.Errorf("rebuild specification index: %w", err)
			}
			if err := insertRevisionReferences(ctx, tx, head); err != nil {
				return 0, err
			}
		} else if existingDirty == 0 {
			number, err := specificationNumber(head.SpecificationID)
			if err != nil {
				return 0, err
			}
			parents, _ := json.Marshal(head.Parents)
			if _, err := tx.ExecContext(ctx, `UPDATE specifications SET id=?,number=?,title=?,status=?,body=?,parents_json=?,
                author_name=?,author_email=?,author_agent=?,updated_at=?,head_hash=?,dirty=0 WHERE uuid=?`,
				head.SpecificationID, number, head.Title, head.Status, head.Content, string(parents),
				head.Author.Name, head.Author.Email, head.Author.Agent, head.CreatedAt.Format(time.RFC3339Nano),
				head.Hash, specUUID); err != nil {
				return 0, fmt.Errorf("refresh specification index: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM specification_references WHERE spec_uuid=?`, specUUID); err != nil {
				return 0, fmt.Errorf("clear reference index: %w", err)
			}
			if err := insertRevisionReferences(ctx, tx, head); err != nil {
				return 0, err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `UPDATE specifications SET head_hash=? WHERE uuid=?`, head.Hash, specUUID); err != nil {
				return 0, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(objects), nil
}

func insertRevisionReferences(ctx context.Context, tx *sql.Tx, revision Revision) error {
	for _, ref := range revision.References {
		if _, err := tx.ExecContext(ctx, `INSERT INTO specification_references
            (id,spec_uuid,kind,path,start_line,end_line,git_commit,created_at) VALUES(?,?,?,?,?,?,?,?)`,
			ref.ID, revision.SpecificationUUID, ref.Kind, ref.Path, ref.StartLine, ref.EndLine,
			nullString(ref.GitCommit), ref.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("rebuild reference index: %w", err)
		}
	}
	return nil
}

func (s *Store) readAllRevisionObjects(projectID string) ([]Revision, error) {
	root := filepath.Join(s.Paths.Objects, "sha256")
	var revisions []Revision
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(relative), "/")
		if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 62 {
			return nil
		}
		revision, err := readRevisionObject(path)
		if err != nil {
			return err
		}
		if revision.Hash != parts[0]+parts[1] {
			return fmt.Errorf("revision object path does not match hash %s", revision.Hash)
		}
		if revision.ProjectID != projectID {
			return fmt.Errorf("revision %s belongs to project %s, not %s", revision.Hash, revision.ProjectID, projectID)
		}
		revisions = append(revisions, revision)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan revision objects: %w", err)
	}
	return revisions, nil
}

func (s *Store) objectPath(hash string) string {
	return filepath.Join(s.Paths.Objects, "sha256", hash[:2], hash[2:])
}

func specificationNumber(id string) (int, error) {
	separator := strings.LastIndexByte(id, '-')
	if separator < 0 {
		return 0, fmt.Errorf("invalid specification ID %q in revision object", id)
	}
	number, err := strconv.Atoi(id[separator+1:])
	if err != nil || number < 1 {
		return 0, fmt.Errorf("invalid specification ID %q in revision object", id)
	}
	return number, nil
}

// GarbageCollect removes only regenerable cache entries. Recorded revision
// objects are intentionally never considered garbage by v0.1.
func (s *Store) GarbageCollect() (int, error) {
	entries, err := os.ReadDir(s.Paths.Cache)
	if err != nil {
		return 0, fmt.Errorf("read cache directory: %w", err)
	}
	removed := 0
	for _, entry := range entries {
		path := filepath.Join(s.Paths.Cache, entry.Name())
		if err := filepath.WalkDir(path, func(_ string, item fs.DirEntry, walkErr error) error {
			if walkErr == nil && !item.IsDir() {
				removed++
			}
			return walkErr
		}); err != nil {
			return removed, fmt.Errorf("inspect cache entry %s: %w", entry.Name(), err)
		}
		if err := os.RemoveAll(path); err != nil {
			return removed, fmt.Errorf("remove cache entry %s: %w", entry.Name(), err)
		}
	}
	return removed, nil
}
