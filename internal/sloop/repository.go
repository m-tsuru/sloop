package sloop

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Store) NextSpecificationNumber(ctx context.Context) (int, error) {
	var next int
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(number), 0) + 1 FROM specifications`).Scan(&next); err != nil {
		return 0, fmt.Errorf("allocate specification ID: %w", err)
	}
	return next, nil
}

func (s *Store) CreateSpecification(ctx context.Context, spec Specification) error {
	parents, err := json.Marshal(spec.Parents)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO specifications
        (uuid,id,number,title,status,body,parents_json,author_name,author_email,author_agent,updated_at,head_hash,dirty)
        VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, spec.UUID, spec.ID, spec.Number, spec.Title, spec.Status,
		normalizeBody(spec.Body), string(parents), spec.Author.Name, spec.Author.Email, spec.Author.Agent,
		spec.UpdatedAt.Format(time.RFC3339Nano), nil, 1)
	if err != nil {
		return fmt.Errorf("create specification: %w", err)
	}
	return nil
}

func scanSpecification(row interface{ Scan(...any) error }) (Specification, error) {
	var spec Specification
	var status, parentsJSON, updated string
	var agent, dirty int
	var head sql.NullString
	err := row.Scan(&spec.UUID, &spec.ID, &spec.Number, &spec.Title, &status, &spec.Body, &parentsJSON,
		&spec.Author.Name, &spec.Author.Email, &agent, &updated, &head, &dirty)
	if err != nil {
		return Specification{}, err
	}
	spec.Status = Status(status)
	spec.Author.Agent = agent != 0
	spec.HeadHash = head.String
	spec.Dirty = dirty != 0
	if err := json.Unmarshal([]byte(parentsJSON), &spec.Parents); err != nil {
		return Specification{}, fmt.Errorf("decode specification parents: %w", err)
	}
	spec.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Specification{}, fmt.Errorf("decode specification timestamp: %w", err)
	}
	return spec, nil
}

const selectSpecification = `SELECT uuid,id,number,title,status,body,parents_json,author_name,author_email,author_agent,updated_at,head_hash,dirty FROM specifications`

func (s *Store) SpecificationByID(ctx context.Context, id string) (Specification, error) {
	spec, err := scanSpecification(s.DB.QueryRowContext(ctx, selectSpecification+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Specification{}, fmt.Errorf("specification %q not found", id)
	}
	if err != nil {
		return Specification{}, fmt.Errorf("load specification: %w", err)
	}
	spec.References, err = s.References(ctx, spec.UUID)
	return spec, err
}

func (s *Store) SpecificationByUUID(ctx context.Context, id string) (Specification, error) {
	spec, err := scanSpecification(s.DB.QueryRowContext(ctx, selectSpecification+` WHERE uuid = ?`, id))
	if err != nil {
		return Specification{}, err
	}
	spec.References, err = s.References(ctx, spec.UUID)
	return spec, err
}

func (s *Store) Specifications(ctx context.Context) ([]Specification, error) {
	rows, err := s.DB.QueryContext(ctx, selectSpecification+` ORDER BY updated_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list specifications: %w", err)
	}
	defer rows.Close()
	var specifications []Specification
	for rows.Next() {
		spec, err := scanSpecification(rows)
		if err != nil {
			return nil, err
		}
		specifications = append(specifications, spec)
	}
	return specifications, rows.Err()
}

func (s *Store) SaveSpecification(ctx context.Context, spec Specification) error {
	parents, err := json.Marshal(spec.Parents)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE specifications SET title=?,status=?,body=?,parents_json=?,
        author_name=?,author_email=?,author_agent=?,updated_at=?,dirty=? WHERE uuid=?`,
		spec.Title, spec.Status, normalizeBody(spec.Body), string(parents), spec.Author.Name, spec.Author.Email,
		spec.Author.Agent, spec.UpdatedAt.Format(time.RFC3339Nano), spec.Dirty, spec.UUID)
	if err != nil {
		return fmt.Errorf("save specification: %w", err)
	}
	return nil
}

func (s *Store) References(ctx context.Context, specUUID string) ([]Reference, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,kind,path,start_line,end_line,git_commit,created_at
        FROM specification_references WHERE spec_uuid=? ORDER BY kind,path,start_line,end_line,id`, specUUID)
	if err != nil {
		return nil, fmt.Errorf("load references: %w", err)
	}
	defer rows.Close()
	var refs []Reference
	for rows.Next() {
		var ref Reference
		var start, end sql.NullInt64
		var commit sql.NullString
		var created string
		if err := rows.Scan(&ref.ID, &ref.Kind, &ref.Path, &start, &end, &commit, &created); err != nil {
			return nil, err
		}
		if start.Valid {
			v := int(start.Int64)
			ref.StartLine = &v
		}
		if end.Valid {
			v := int(end.Int64)
			ref.EndLine = &v
		}
		ref.GitCommit = commit.String
		ref.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

type canonicalRevision struct {
	ProjectID            string               `json:"project_id"`
	SpecificationUUID    string               `json:"specification_uuid"`
	SpecificationID      string               `json:"specification_id"`
	ParentRevisionHashes []string             `json:"parent_revision_hashes"`
	Author               Author               `json:"author"`
	Title                string               `json:"title"`
	Content              string               `json:"content"`
	Status               Status               `json:"status"`
	Parents              []string             `json:"parents"`
	References           []canonicalReference `json:"references"`
}

type canonicalReference struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
}

func revisionHash(revision Revision) (string, error) {
	parents := append([]string(nil), revision.ParentRevisionHashes...)
	sort.Strings(parents)
	refs := make([]canonicalReference, 0, len(revision.References))
	for _, ref := range revision.References {
		refs = append(refs, canonicalReference{ref.ID, ref.Kind, ref.Path, ref.StartLine, ref.EndLine, ref.GitCommit})
	}
	sort.Slice(refs, func(i, j int) bool {
		a, _ := json.Marshal(refs[i])
		b, _ := json.Marshal(refs[j])
		return string(a) < string(b)
	})
	canonical := canonicalRevision{
		ProjectID: revision.ProjectID, SpecificationUUID: revision.SpecificationUUID,
		SpecificationID: revision.SpecificationID, ParentRevisionHashes: parents,
		Author: revision.Author, Title: revision.Title, Content: normalizeBody(revision.Content),
		Status: revision.Status, Parents: revision.Parents, References: refs,
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode canonical revision: %w", err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (s *Store) RecordRevision(ctx context.Context, projectID string, spec *Specification, author Author) (Revision, bool, error) {
	if !spec.Dirty && spec.HeadHash != "" {
		revision, err := s.RevisionByHash(ctx, spec.HeadHash)
		return revision, false, err
	}
	refs, err := s.References(ctx, spec.UUID)
	if err != nil {
		return Revision{}, false, err
	}
	var revisionNumber int
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision_number),0)+1 FROM revision_index WHERE spec_uuid=?`, spec.UUID).Scan(&revisionNumber); err != nil {
		return Revision{}, false, fmt.Errorf("allocate revision number: %w", err)
	}
	parentHashes := []string{}
	if spec.HeadHash != "" {
		parentHashes = append(parentHashes, spec.HeadHash)
	}
	now := time.Now().Truncate(time.Microsecond)
	revision := Revision{
		FormatVersion: 1, ProjectID: projectID, SpecificationUUID: spec.UUID, SpecificationID: spec.ID,
		RevisionNumber: revisionNumber, ParentRevisionHashes: parentHashes, Author: author,
		Title: spec.Title, Content: normalizeBody(spec.Body), Status: spec.Status,
		Parents: append([]string(nil), spec.Parents...), References: refs, CreatedAt: now,
	}
	revision.Hash, err = revisionHash(revision)
	if err != nil {
		return Revision{}, false, err
	}
	objectPath, err := s.writeRevisionObject(revision)
	if err != nil {
		return Revision{}, false, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Revision{}, false, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO revision_index
        (hash,spec_uuid,spec_id,revision_number,status,title,author_name,author_email,author_agent,created_at,object_path)
        VALUES(?,?,?,?,?,?,?,?,?,?,?)`, revision.Hash, spec.UUID, spec.ID, revisionNumber, revision.Status,
		revision.Title, author.Name, author.Email, author.Agent, now.Format(time.RFC3339Nano), objectPath)
	if err != nil {
		return Revision{}, false, fmt.Errorf("index revision: %w", err)
	}
	for _, parent := range parentHashes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO revision_parents(revision_hash,parent_hash) VALUES(?,?)`, revision.Hash, parent); err != nil {
			return Revision{}, false, fmt.Errorf("index revision parent: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE specifications SET head_hash=?,dirty=0 WHERE uuid=?`, revision.Hash, spec.UUID); err != nil {
		return Revision{}, false, fmt.Errorf("advance specification head: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Revision{}, false, fmt.Errorf("commit revision: %w", err)
	}
	spec.HeadHash = revision.Hash
	spec.Dirty = false
	spec.References = refs
	return revision, true, nil
}

func (s *Store) writeRevisionObject(revision Revision) (string, error) {
	if len(revision.Hash) != 64 {
		return "", errors.New("revision object has invalid hash")
	}
	dir := filepath.Join(s.Paths.Objects, "sha256", revision.Hash[:2])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create revision object directory: %w", err)
	}
	path := filepath.Join(dir, revision.Hash[2:])
	data, err := json.MarshalIndent(revision, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode revision object: %w", err)
	}
	data = append(data, '\n')
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) != string(data) {
			return "", fmt.Errorf("object collision for revision %s", revision.Hash)
		}
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect revision object: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".revision-*")
	if err != nil {
		return "", fmt.Errorf("create revision object: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write revision object: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("sync revision object: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close revision object: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", fmt.Errorf("install revision object: %w", err)
	}
	return path, nil
}

func (s *Store) RevisionByHash(ctx context.Context, hash string) (Revision, error) {
	var objectPath string
	if err := s.DB.QueryRowContext(ctx, `SELECT object_path FROM revision_index WHERE hash=?`, hash).Scan(&objectPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Revision{}, fmt.Errorf("revision %q not found", hash)
		}
		return Revision{}, err
	}
	return readRevisionObject(objectPath)
}

func readRevisionObject(path string) (Revision, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Revision{}, fmt.Errorf("read revision object: %w", err)
	}
	var revision Revision
	if err := json.Unmarshal(data, &revision); err != nil {
		return Revision{}, fmt.Errorf("decode revision object: %w", err)
	}
	expected, err := revisionHash(revision)
	if err != nil {
		return Revision{}, err
	}
	if expected != revision.Hash {
		return Revision{}, fmt.Errorf("revision object %s failed hash verification", revision.Hash)
	}
	return revision, nil
}

func (s *Store) Revisions(ctx context.Context, specUUID string) ([]RevisionSummary, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT hash,spec_uuid,spec_id,revision_number,status,title,
        author_name,author_email,author_agent,created_at,object_path FROM revision_index
        WHERE spec_uuid=? ORDER BY revision_number`, specUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var revisions []RevisionSummary
	for rows.Next() {
		var revision RevisionSummary
		var status, created string
		var agent int
		if err := rows.Scan(&revision.Hash, &revision.SpecUUID, &revision.SpecID, &revision.RevisionNumber,
			&status, &revision.Title, &revision.Author.Name, &revision.Author.Email, &agent, &created, &revision.ObjectPath); err != nil {
			return nil, err
		}
		revision.Status = Status(status)
		revision.Author.Agent = agent != 0
		revision.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		revisions = append(revisions, revision)
	}
	return revisions, rows.Err()
}

func (s *Store) RevisionByNumber(ctx context.Context, specUUID string, number int) (Revision, error) {
	var hash string
	if err := s.DB.QueryRowContext(ctx, `SELECT hash FROM revision_index WHERE spec_uuid=? AND revision_number=?`, specUUID, number).Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Revision{}, fmt.Errorf("revision %d not found", number)
		}
		return Revision{}, err
	}
	return s.RevisionByHash(ctx, hash)
}

func (s *Store) ResolveRevisionPrefix(ctx context.Context, prefix string) (Revision, error) {
	if len(prefix) < 4 || len(prefix) > 64 {
		return Revision{}, fmt.Errorf("invalid revision hash prefix %q", prefix)
	}
	if _, err := hex.DecodeString(prefix); err != nil {
		return Revision{}, fmt.Errorf("invalid revision hash prefix %q", prefix)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT hash FROM revision_index WHERE hash LIKE ? ORDER BY hash LIMIT 2`, strings.ToLower(prefix)+"%")
	if err != nil {
		return Revision{}, err
	}
	defer rows.Close()
	var hashes []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return Revision{}, err
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 {
		return Revision{}, fmt.Errorf("revision prefix %q not found", prefix)
	}
	if len(hashes) > 1 {
		return Revision{}, fmt.Errorf("revision prefix '%s' is ambiguous.\nPlease provide a longer revision hash", prefix)
	}
	return s.RevisionByHash(ctx, hashes[0])
}

func (s *Store) RestoreRevision(ctx context.Context, projectID string, spec *Specification, source Revision, author Author) (Revision, error) {
	spec.Title = source.Title
	spec.Status = source.Status
	spec.Body = source.Content
	spec.Parents = append([]string(nil), source.Parents...)
	spec.Author = author
	spec.UpdatedAt = time.Now().Truncate(time.Microsecond)
	spec.Dirty = true
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Revision{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM specification_references WHERE spec_uuid=?`, spec.UUID); err != nil {
		tx.Rollback()
		return Revision{}, err
	}
	for _, ref := range source.References {
		if _, err := tx.ExecContext(ctx, `INSERT INTO specification_references
            (id,spec_uuid,kind,path,start_line,end_line,git_commit,created_at) VALUES(?,?,?,?,?,?,?,?)`,
			ref.ID, spec.UUID, ref.Kind, ref.Path, ref.StartLine, ref.EndLine, nullString(ref.GitCommit), ref.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			tx.Rollback()
			return Revision{}, err
		}
	}
	parents, _ := json.Marshal(spec.Parents)
	if _, err := tx.ExecContext(ctx, `UPDATE specifications SET title=?,status=?,body=?,parents_json=?,author_name=?,author_email=?,author_agent=?,updated_at=?,dirty=1 WHERE uuid=?`,
		spec.Title, spec.Status, spec.Body, string(parents), author.Name, author.Email, author.Agent,
		spec.UpdatedAt.Format(time.RFC3339Nano), spec.UUID); err != nil {
		tx.Rollback()
		return Revision{}, err
	}
	if err := tx.Commit(); err != nil {
		return Revision{}, err
	}
	spec.References = source.References
	revision, _, err := s.RecordRevision(ctx, projectID, spec, author)
	return revision, err
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func ParseSpecificationNumber(value string) (int, bool) {
	n, err := strconv.Atoi(value)
	return n, err == nil && n > 0
}

func NewSpecification(prefix string, number int, author Author, body string) Specification {
	return Specification{
		UUID: uuid.NewString(), ID: fmt.Sprintf("%s-%d", prefix, number), Number: number,
		Status: StatusDraft, Body: normalizeBody(body), Parents: []string{}, Author: author,
		UpdatedAt: time.Now().Truncate(time.Microsecond), Dirty: true,
	}
}

func (s *Store) RecordStatusTransition(ctx context.Context, transition StatusTransition) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO status_transitions
        (spec_uuid,based_on_revision,resulting_revision,status,reason,author_name,author_email,author_agent,created_at)
        VALUES(?,?,?,?,?,?,?,?,?)`, transition.SpecUUID, nullString(transition.BasedOnRevision),
		transition.ResultingRevision, transition.Status, transition.Reason, transition.Author.Name,
		transition.Author.Email, transition.Author.Agent, transition.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record status transition: %w", err)
	}
	return nil
}

func (s *Store) AddReview(ctx context.Context, review Review) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO reviews
        (spec_uuid,revision_hash,result,reason,author_name,author_email,author_agent,created_at)
        VALUES(?,?,?,?,?,?,?,?)`, review.SpecUUID, review.RevisionHash, review.Result, review.Reason,
		review.Author.Name, review.Author.Email, review.Author.Agent, review.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record review result: %w", err)
	}
	return nil
}

func (s *Store) Reviews(ctx context.Context, specUUID string) ([]Review, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,spec_uuid,revision_hash,result,reason,
        author_name,author_email,author_agent,created_at FROM reviews WHERE spec_uuid=? ORDER BY id`, specUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reviews []Review
	for rows.Next() {
		var review Review
		var agent int
		var created string
		if err := rows.Scan(&review.ID, &review.SpecUUID, &review.RevisionHash, &review.Result, &review.Reason,
			&review.Author.Name, &review.Author.Email, &agent, &created); err != nil {
			return nil, err
		}
		review.Author.Agent = agent != 0
		review.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (s *Store) AddAgentRun(ctx context.Context, run AgentRun) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO agent_runs
        (spec_uuid,revision_hash,result,reason,author_name,author_email,created_at)
        VALUES(?,?,?,?,?,?,?)`, run.SpecUUID, run.RevisionHash, run.Result, run.Reason,
		run.Author.Name, run.Author.Email, run.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record agent run: %w", err)
	}
	return nil
}

func (s *Store) AgentRuns(ctx context.Context, specUUID string) ([]AgentRun, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,spec_uuid,revision_hash,result,reason,
        author_name,author_email,created_at FROM agent_runs WHERE spec_uuid=? ORDER BY id`, specUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []AgentRun
	for rows.Next() {
		var run AgentRun
		var created string
		if err := rows.Scan(&run.ID, &run.SpecUUID, &run.RevisionHash, &run.Result, &run.Reason,
			&run.Author.Name, &run.Author.Email, &created); err != nil {
			return nil, err
		}
		run.Author.Agent = true
		run.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *Store) AddReference(ctx context.Context, spec *Specification, ref Reference, author Author) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO specification_references
        (id,spec_uuid,kind,path,start_line,end_line,git_commit,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		ref.ID, spec.UUID, ref.Kind, ref.Path, ref.StartLine, ref.EndLine, nullString(ref.GitCommit),
		ref.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("add reference: %w", err)
	}
	if spec.Status != StatusDraft {
		spec.Status = StatusDraft
	}
	spec.Author = author
	spec.UpdatedAt = ref.CreatedAt
	spec.Dirty = true
	if _, err := tx.ExecContext(ctx, `UPDATE specifications SET status=?,author_name=?,author_email=?,author_agent=?,updated_at=?,dirty=1 WHERE uuid=?`,
		spec.Status, author.Name, author.Email, author.Agent, spec.UpdatedAt.Format(time.RFC3339Nano), spec.UUID); err != nil {
		return fmt.Errorf("mark specification changed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	spec.References = append(spec.References, ref)
	return nil
}

func (s *Store) RemoveReference(ctx context.Context, spec *Specification, idPrefix string, author Author) (Reference, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,kind,path,start_line,end_line,git_commit,created_at
        FROM specification_references WHERE spec_uuid=? AND id LIKE ? ORDER BY id LIMIT 2`, spec.UUID, idPrefix+"%")
	if err != nil {
		return Reference{}, err
	}
	var matches []Reference
	for rows.Next() {
		var ref Reference
		var start, end sql.NullInt64
		var commit sql.NullString
		var created string
		if err := rows.Scan(&ref.ID, &ref.Kind, &ref.Path, &start, &end, &commit, &created); err != nil {
			rows.Close()
			return Reference{}, err
		}
		if start.Valid {
			value := int(start.Int64)
			ref.StartLine = &value
		}
		if end.Valid {
			value := int(end.Int64)
			ref.EndLine = &value
		}
		ref.GitCommit = commit.String
		ref.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		matches = append(matches, ref)
	}
	if err := rows.Close(); err != nil {
		return Reference{}, err
	}
	if len(matches) == 0 {
		return Reference{}, fmt.Errorf("reference %q not found for %s", idPrefix, spec.ID)
	}
	if len(matches) > 1 {
		return Reference{}, fmt.Errorf("reference prefix %q is ambiguous", idPrefix)
	}

	now := time.Now().Truncate(time.Microsecond)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Reference{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM specification_references WHERE id=? AND spec_uuid=?`, matches[0].ID, spec.UUID); err != nil {
		return Reference{}, fmt.Errorf("remove reference: %w", err)
	}
	if spec.Status != StatusDraft {
		spec.Status = StatusDraft
	}
	spec.Author = author
	spec.UpdatedAt = now
	spec.Dirty = true
	if _, err := tx.ExecContext(ctx, `UPDATE specifications SET status=?,author_name=?,author_email=?,author_agent=?,updated_at=?,dirty=1 WHERE uuid=?`,
		spec.Status, author.Name, author.Email, author.Agent, now.Format(time.RFC3339Nano), spec.UUID); err != nil {
		return Reference{}, fmt.Errorf("mark specification changed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Reference{}, err
	}
	for i := range spec.References {
		if spec.References[i].ID == matches[0].ID {
			spec.References = append(spec.References[:i], spec.References[i+1:]...)
			break
		}
	}
	return matches[0], nil
}

func (s *Store) GitRelations(ctx context.Context, specUUID string) ([]GitRelation, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT ri.spec_id,gr.revision_hash,gr.git_commit,gr.relation,gr.created_at
        FROM git_relations gr JOIN revision_index ri ON ri.hash=gr.revision_hash
        WHERE gr.spec_uuid=? ORDER BY gr.created_at,gr.git_commit`, specUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var relations []GitRelation
	for rows.Next() {
		var relation GitRelation
		var created string
		if err := rows.Scan(&relation.SpecificationID, &relation.RevisionHash, &relation.Commit, &relation.Relation, &created); err != nil {
			return nil, err
		}
		relation.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		relations = append(relations, relation)
	}
	return relations, rows.Err()
}

func (s *Store) AddGitRelation(ctx context.Context, specUUID string, relation GitRelation) error {
	_, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO git_relations
        (spec_uuid,revision_hash,git_commit,relation,created_at) VALUES(?,?,?,?,?)`,
		specUUID, relation.RevisionHash, relation.Commit, relation.Relation,
		relation.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record Git relation: %w", err)
	}
	return nil
}
