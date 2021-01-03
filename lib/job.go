package lib

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/thoas/go-funk"
)

// CheckSuite represents a check suite notified by GitHub.
type CheckSuite struct {
	ID                   int       `db:"id"`
	GitHubID             int64     `db:"github_id"`
	Status               string    `db:"status"`
	GitHubInstallationID int64     `db:"github_installation_id"`
	RepoName             string    `db:"repo_name"`
	RepoOwner            string    `db:"repo_owner"`
	RepoCloneURL         string    `db:"repo_clone_url"`
	CommitRef            string    `db:"commit_ref"`
	Created              time.Time `db:"created"`
}

// Job represents the work that a single runner can do within a single CI workflow.
type Job struct {
	ID               int                    `db:"id"`
	CheckSuiteID     int                    `db:"check_suite"`
	GitHubCheckRunID int64                  `db:"github_check_run_id"`
	Status           string                 `db:"status"`
	GitHubStatus     *string                `db:"github_status"`
	Runner           *string                `db:"runner"`
	SkippedByRunners pq.StringArray         `db:"skipped_by_runners"`
	TestRequirements TestRequirementsMatrix `db:"test_requirements"`
	Tests            Tests                  `db:"test_results"`
	Start            *time.Time             `db:"ts_start"`
	End              *time.Time             `db:"ts_end"`
}

type Tests []Test

func (a Tests) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Tests) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type TestRequirementsMatrix struct {
	Original  TestRequirements // the test requirements as parsed from the original test
	Effective TestRequirements // the test requirements to use (with potentially more requirements)
}

func (a TestRequirementsMatrix) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *TestRequirementsMatrix) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// Name returns a formatted name representing this job.
func (j *Job) Name() string {
	var tokens []string
	for _, s := range j.TestRequirements.Effective.Sketches {
		var tags []string
		if s.RequireFQBN != "" && s.RequireFQBN != "*" {
			tags = append(tags, s.RequireFQBN)
		} else if s.RequireArchitecture != "" && s.RequireArchitecture != "*" {
			tags = append(tags, s.RequireArchitecture)
		}
		tags = append(tags, s.RequireFeatures...)
		if len(tags) > 0 {
			tokens = append(tokens, strings.Join(tags, ","))
		}
	}
	if len(j.TestRequirements.Effective.RequireWiring) > 0 {
		tokens = append(tokens, strings.Join(j.TestRequirements.Effective.RequireWiring, ","))
	}
	return "Hardware test: " + strings.Join(tokens, " ")
}

func (j *Job) DeviceFQBNs() (out []string) {
	for _, t := range j.Tests {
		out = append(out, t.DeviceFQBNs...)
	}
	return funk.UniqString(out)
}

func (j *Job) StatusFromResults() string {
	status := "skipped"
	for _, t := range j.Tests {
		if status == "skipped" && t.Status == "success" {
			status = "success"
		} else if t.Status == "failure" {
			status = "failure"
		}
	}
	return status
}

func (j *Job) Report() (out string) {
	if j.Tests == nil {
		return ""
	}
	for _, t := range j.Tests {
		out += fmt.Sprintf("Running test in %s:\n", t.RelPath())
		out += t.Output
		out += "\n"
	}
	return out
}
