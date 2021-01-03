package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	. "github.com/alranel/cino/lib"
	"github.com/google/go-github/github"
	"github.com/lib/pq"
)

func StartResultsHandler() {
	var runnerIDs []string
	for _, r := range Config.Runners {
		runnerIDs = append(runnerIDs, r.ID)
	}

	ListenChannel(Config.DB, "changed_jobs", func(*pq.Notification) {
		db := ConnectDB(Config.DB)

		for {
			tx := db.MustBegin()
			var job Job
			err := tx.Get(&job, `select * from jobs 
				where (status = 'queued' AND skipped_by_runners @> $1)
				or (status IN ('success', 'failure', 'in_progress') AND github_status != status)
				order by id for update limit 1`,
				pq.Array(runnerIDs))
			if err == sql.ErrNoRows {
				break
			} else if err != nil {
				panic(err)
			}
			fmt.Printf("Processing results for job %d\n", job.ID)

			if job.Status == "queued" {
				job.Status = "skipped"
			}
			job.GitHubStatus = &job.Status
			tx.NamedExec(`update jobs set status = :status, github_status = :github_status
				where id = :id`, &job)

			var checkSuite CheckSuite
			tx.Get(&checkSuite, `select * from check_suites where id = $1`, job.CheckSuiteID)

			ctx := context.Background()
			ghClient := GitHubClient(checkSuite.GitHubInstallationID)
			checkRunOpts := github.UpdateCheckRunOptions{
				Name:       job.Name(),
				ExternalID: github.String(fmt.Sprint(job.ID)),
			}
			if *job.GitHubStatus == "in_progress" {
				checkRunOpts.Status = job.GitHubStatus
			} else if *job.GitHubStatus == "success" || *job.GitHubStatus == "failure" || *job.GitHubStatus == "skipped" {
				checkRunOpts.Status = github.String("completed")
				checkRunOpts.Conclusion = job.GitHubStatus
			}
			if job.Status == "skipped" {
				checkRunOpts.Output = new(github.CheckRunOutput)
				checkRunOpts.Output.Title = github.String("No suitable device")
				summary := "No suitable runners matching the following features:\n\n"
				if len(job.TestRequirements.Effective.RequireWiring) > 0 {
					summary += fmt.Sprintf("* %s\n", strings.Join(job.TestRequirements.Effective.RequireWiring, ", "))
				}
				for _, s := range job.TestRequirements.Effective.Sketches {
					summary += "* Device:\n"
					if s.RequireArchitecture != "" {
						summary += fmt.Sprintf("   * Architecture: %s\n", s.RequireArchitecture)
					}
					if s.RequireFQBN != "" {
						summary += fmt.Sprintf("   * Board: %s\n", s.RequireFQBN)
					}
					if len(s.RequireFeatures) > 0 {
						summary += fmt.Sprintf("   * Features: %s\n", strings.Join(s.RequireFeatures, ", "))
					}
				}
				checkRunOpts.Output.Summary = github.String(summary)
			} else if len(job.Tests) > 0 {
				checkRunOpts.Output = new(github.CheckRunOutput)
				if job.Status == "success" {
					checkRunOpts.Output.Title = github.String("All tests passed")
				} else {
					checkRunOpts.Output.Title = github.String("Tests failed")
				}
				summary := fmt.Sprintf("%d test(s) were run:\n\n", len(job.Tests))
				for _, t := range job.Tests {
					summary += fmt.Sprintf("* `%s`\n", t.RelPath())
				}
				summary += fmt.Sprintf("\nusing the following board(s) attached to **%s**:\n\n", *job.Runner)
				for _, d := range job.DeviceFQBNs() {
					summary += fmt.Sprintf("* %s\n", d)
				}
				checkRunOpts.Output.Summary = github.String(summary)
				checkRunOpts.Output.Text = github.String(job.Report())
			}
			if job.End != nil {
				checkRunOpts.CompletedAt = &github.Timestamp{Time: *job.End}
			}
			_, _, err = ghClient.Checks.UpdateCheckRun(ctx,
				checkSuite.RepoOwner, checkSuite.RepoName, job.GitHubCheckRunID, checkRunOpts)
			if err != nil {
				panic(err)
			}

			tx.Commit()
		}
	})
}
