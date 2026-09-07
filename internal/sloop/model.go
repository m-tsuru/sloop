package sloop

import (
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusDraft       Status = "DRAFT"
	StatusReady       Status = "READY"
	StatusForceReady  Status = "FORCEREADY"
	StatusImplemented Status = "IMPLEMENTED"
	StatusVerified    Status = "VERIFIED"
	StatusCanceled    Status = "CANCELED"
	StatusCompleted   Status = "COMPLETED"
)

var validStatuses = map[Status]bool{
	StatusDraft: true, StatusReady: true, StatusForceReady: true,
	StatusImplemented: true, StatusVerified: true, StatusCanceled: true,
	StatusCompleted: true,
}

func ParseStatus(value string) (Status, error) {
	status := Status(strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "-", "")))
	if !validStatuses[status] {
		return "", fmt.Errorf("invalid specification status %q", value)
	}
	return status, nil
}

type Author struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
	Agent bool   `json:"agent" yaml:"agent"`
}

func (a Author) Equal(other Author) bool {
	return a.Name == other.Name && a.Email == other.Email && a.Agent == other.Agent
}

type Reference struct {
	ID        string    `json:"id" yaml:"id"`
	Kind      string    `json:"kind" yaml:"kind"`
	Path      string    `json:"path" yaml:"path"`
	StartLine *int      `json:"start_line,omitempty" yaml:"start-line,omitempty"`
	EndLine   *int      `json:"end_line,omitempty" yaml:"end-line,omitempty"`
	GitCommit string    `json:"git_commit,omitempty" yaml:"git-commit,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"created-at"`
}

type Specification struct {
	UUID       string
	ID         string
	Number     int
	Title      string
	Status     Status
	Body       string
	Parents    []string
	Author     Author
	UpdatedAt  time.Time
	HeadHash   string
	Dirty      bool
	References []Reference
}

type Revision struct {
	FormatVersion        int         `json:"format_version"`
	Hash                 string      `json:"hash"`
	ProjectID            string      `json:"project_id"`
	SpecificationUUID    string      `json:"specification_uuid"`
	SpecificationID      string      `json:"specification_id"`
	RevisionNumber       int         `json:"revision_number"`
	ParentRevisionHashes []string    `json:"parent_revision_hashes"`
	Author               Author      `json:"author"`
	Title                string      `json:"title"`
	Content              string      `json:"content"`
	Status               Status      `json:"status"`
	Parents              []string    `json:"parents"`
	References           []Reference `json:"references"`
	CreatedAt            time.Time   `json:"created_at"`
}

type RevisionSummary struct {
	Hash           string
	SpecUUID       string
	SpecID         string
	RevisionNumber int
	Status         Status
	Title          string
	Author         Author
	CreatedAt      time.Time
	ObjectPath     string
}

type Review struct {
	ID           int64     `json:"id" yaml:"id"`
	SpecUUID     string    `json:"-" yaml:"-"`
	RevisionHash string    `json:"revision_hash" yaml:"revision-hash"`
	Result       string    `json:"result" yaml:"result"`
	Reason       string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Author       Author    `json:"author" yaml:"author"`
	CreatedAt    time.Time `json:"created_at" yaml:"date"`
}

type AgentRun struct {
	ID           int64     `json:"id" yaml:"id"`
	SpecUUID     string    `json:"-" yaml:"-"`
	RevisionHash string    `json:"revision_hash" yaml:"revision-hash"`
	Result       string    `json:"result" yaml:"result"`
	Reason       string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Author       Author    `json:"author" yaml:"author"`
	CreatedAt    time.Time `json:"created_at" yaml:"date"`
}

type StatusTransition struct {
	ID                int64
	SpecUUID          string
	BasedOnRevision   string
	ResultingRevision string
	Status            Status
	Reason            string
	Author            Author
	CreatedAt         time.Time
}

type GitRelation struct {
	SpecificationID string    `json:"specification_id" yaml:"id"`
	RevisionHash    string    `json:"revision_hash" yaml:"revision-hash"`
	Commit          string    `json:"commit" yaml:"commit"`
	Relation        string    `json:"relation" yaml:"relation"`
	CreatedAt       time.Time `json:"created_at,omitempty" yaml:"date,omitempty"`
}
