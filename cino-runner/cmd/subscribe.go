package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/alranel/cino/cino-runner/runner"
	. "github.com/alranel/cino/lib"
	"github.com/lib/pq"
	"github.com/spf13/cobra"
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribes to a cino-server instance and waits for jobs",
	Long:  `This command starts the runner in client mode. It subscribes to a central cino-server instance and waits for jobs to execute.`,
	Run:   runSubscribe,
}

func init() {
	rootCmd.AddCommand(subscribeCmd)
}

func runSubscribe(cmd *cobra.Command, args []string) {
	if runner.Config.RunnerID == "" {
		log.Fatal("runner_id not configured")
	}
	if runner.Config.DB.DSN == "" {
		log.Fatal("Database DSN not configured")
	}
	if len(runner.Config.Devices) == 0 {
		log.Fatal("No devices configured")
	}
	for _, device := range runner.Config.Devices {
		if _, err := os.Stat(device.Port); os.IsNotExist(err) {
			log.Fatalf("Device %s not found\n", device.Port)
		}
	}

	fmt.Printf("Waiting for jobs...\n")
	ListenChannel(runner.Config.DB, "new_jobs", func(*pq.Notification) {
		db := ConnectDB(runner.Config.DB)

		for {
			tx := db.MustBegin()
			job := Job{}
			err := tx.Get(&job, `select * from jobs 
				where (status = 'queued') or (status = 'in_progress' and runner = $1) 
				and not $1 = any(skipped_by_runners)
				order by id for update limit 1`,
				runner.Config.RunnerID)
			if err == sql.ErrNoRows {
				break
			} else if err != nil {
				panic(err)
			}

			fmt.Printf("Processing job %d\n", job.ID)

			// Try to assign devices
			devices := runner.AssignDevices(job.TestRequirements.Effective)
			if devices == nil {
				// We can't run this job, skip it
				tx.MustExec("update jobs set skipped_by_runners = array_append(skipped_by_runners, $1) where id = $2",
					runner.Config.RunnerID, job.ID)
				tx.Commit()
				continue
			}

			// Okay, we can run this job
			tx.MustExec("update jobs set status = 'in_progress', runner = $1, ts_start = now() where id = $2",
				runner.Config.RunnerID, job.ID)
			tx.Commit()

			var checkSuite CheckSuite
			err = db.Get(&checkSuite, `select * from check_suites where id = $1`, job.CheckSuiteID)
			if err != nil {
				panic(err)
			}

			repoDir, err := CloneRepo(checkSuite.RepoCloneURL, checkSuite.CommitRef)
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(repoDir)

			// Look for tests
			{
				tt, err := FindTests(repoDir)
				if err != nil {
					panic(err)
				}
				job.Tests = tt
			}

			// If no tests, report skipped status
			// (we should never get here though)
			if len(job.Tests) == 0 {
				db.MustExec("update jobs set skipped_by_runners = array_append(skipped_by_runners, $1) where id = $2",
					runner.Config.RunnerID, job.ID)
				continue
			}

			// Run tests
			for i := range job.Tests {
				if !job.Tests[i].GetRequirements().Equals(job.TestRequirements.Original) {
					fmt.Printf("  skipping test %d having other job requirements\n", i)
					continue
				}
				err = runner.RunTest(&job.Tests[i], devices)
				if err != nil {
					panic(err)
				}
			}
			fmt.Printf("Job completed\n")

			// Update job
			status := job.StatusFromResults()
			if status == "skipped" {
				db.MustExec("update jobs set status = 'queued', skipped_by_runners = array_append(skipped_by_runners, $1) where id = $2",
					runner.Config.RunnerID, job.ID)
			} else if status == "success" || status == "failure" {
				db.MustExec("update jobs set status = $1, test_results = $2, ts_end = now() where id = $3",
					status, job.Tests, job.ID)
			}
		}
	})
}
