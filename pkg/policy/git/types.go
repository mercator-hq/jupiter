package git

import (
	"time"
)

// CommitInfo contains metadata about a Git commit.
type CommitInfo struct {
	SHA        string    `json:"sha"`
	Author     string    `json:"author"`
	Email      string    `json:"email"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	Branch     string    `json:"branch"`
	Repository string    `json:"repository"`
}

// PullResult contains result of a pull operation.
type PullResult struct {
	FromSHA      string
	ToSHA        string
	ChangedFiles []string
	HadChanges   bool
}

// RepositoryMetrics tracks Git operation metrics.
type RepositoryMetrics struct {
	CloneDuration   time.Duration
	PullDuration    time.Duration
	LastCommitSHA   string
	LastPullTime    time.Time
	FailedPulls     int64
	SuccessfulPulls int64
}

// CommitHistory tracks policy version history.
type CommitHistory struct {
	Current  *CommitInfo   `json:"current"`
	Previous *CommitInfo   `json:"previous,omitempty"`
	History  []*CommitInfo `json:"history"` // Last N commits
}
