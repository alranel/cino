package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/alranel/cino/lib"
	"github.com/google/go-github/github"
	"github.com/lib/pq"
	"github.com/thoas/go-funk"
	"gopkg.in/ini.v1"
)

// StartScanner listens to a queue containing incoming check_suite notifications
// from GitHub. For each newly created check_suite, the repository is cloned locally
// and inspected to find runnable tests. A job matrix is then generated according to
// the repository type (core, library, sketch) and its properties.
func StartScanner() {
	ListenChannel(Config.DB, "new_check_suites", func(*pq.Notification) {
		db := ConnectDB(Config.DB)

		checkSuites := []CheckSuite{}
		err := db.Select(&checkSuites, `select * from check_suites 
			where status = 'pending' order by id`)
		if err != nil {
			panic(err)
		}

		for _, checkSuite := range checkSuites {
			fmt.Printf("Processing GitHub check suite %d\n", checkSuite.ID)

			// Clone repo
			repoDir, err := CloneRepo(checkSuite.RepoCloneURL, checkSuite.CommitRef)
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(repoDir)

			// Look for tests
			tests, err := FindTests(repoDir)
			if err != nil {
				panic(err)
			}

			// Extract requirements from tests
			requirements := make([]TestRequirements, 0, len(tests))
			for _, test := range tests {
				requirements = append(requirements, test.GetRequirements())
			}

			var matrix []TestRequirementsMatrix
			if IsLibrary(repoDir) {
				// Get all architectures supported by this library.
				architectures, err := getArchitecturesFromLibrary(repoDir)
				if err != nil {
					fmt.Printf("failed to get architectures for library at %s\n", checkSuite.RepoCloneURL)
					continue
				}
				if len(architectures) == 1 && architectures[0] == "*" {
					// Use the architectures list from our configuration file
					architectures = Config.Architectures
				} else if len(Config.Architectures) > 0 {
					// Limit the list to the ones we have in our configuration file
					architectures = funk.IntersectString(architectures, Config.Architectures)
				}

				// Repeat the entire test set for each architecture
				matrix = RepeatByArchitectures(requirements, architectures)
			} else if IsCore(repoDir) {
				// Get all FQBNs supported by this core.
				fqbns, err := getBoardsFromCore(repoDir)
				if err != nil {
					fmt.Printf("failed to get boards for core at %s\n", checkSuite.RepoCloneURL)
					continue
				}

				// Repeat the entire test set for each FQBN
				matrix = RepeatByFQBNs(requirements, fqbns)
			} else {
				for _, tr := range requirements {
					matrix = append(matrix, TestRequirementsMatrix{
						Original:  tr,
						Effective: tr,
					})
				}
			}

			// Remove duplicates
			matrix = uniqRequirements(matrix)

			// Store jobs and notify runners
			ctx := context.Background()
			ghClient := GitHubClient(checkSuite.GitHubInstallationID)
			tx := db.MustBegin()
			tx.MustExec(`UPDATE check_suites SET status = 'dispatched' WHERE id = $1`, checkSuite.ID)
			for _, r := range matrix {
				queued := "queued"
				job := Job{
					CheckSuiteID:     checkSuite.ID,
					Status:           "queued",
					GitHubStatus:     &queued,
					TestRequirements: r,
				}

				// Create check run in GitHub
				checkrun, _, err := ghClient.Checks.CreateCheckRun(ctx,
					checkSuite.RepoOwner, checkSuite.RepoName, github.CreateCheckRunOptions{
						Name:    job.Name(),
						HeadSHA: checkSuite.CommitRef,
						Status:  job.GitHubStatus,
					})
				if err != nil {
					panic(err)
				}
				job.GitHubCheckRunID = *checkrun.ID
				fmt.Printf("  created GitHub check run %d (%s)\n", checkrun.ID, job.Name())

				// Store in database
				_, err = tx.NamedExec(`INSERT INTO jobs 
					(check_suite, github_check_run_id, status, github_status, test_requirements) 
					VALUES (:check_suite, :github_check_run_id, :status, :github_status, :test_requirements)`,
					&job)
				if err != nil {
					panic(err)
				}
			}
			tx.Commit()
		}
	})
}

func getArchitecturesFromLibrary(dir string) ([]string, error) {
	f, err := ini.Load(filepath.Join(dir, "library.properties"))
	if err != nil {
		return nil, err
	}
	return strings.Split(f.Section("").Key("architectures").String(), ","), nil
}

func getBoardsFromCore(dir string) ([]string, error) {
	f, err := ini.Load(filepath.Join(dir, "boards.txt"))
	if err != nil {
		return nil, err
	}

	var out []string
	re := regexp.MustCompile(`^([^.]+)\.name$`)
	for _, k := range f.Section("").KeyStrings() {
		res := re.FindStringSubmatch(k)
		if len(res) == 2 {
			out = append(out, res[1])
		}
	}
	return out, nil
}

func RepeatByArchitectures(tmpl []TestRequirements, architectures []string) []TestRequirementsMatrix {
	out := make([]TestRequirementsMatrix, 0, len(tmpl)*len(architectures))
	for _, r := range tmpl {
		combinations := Perm(architectures, len(r.Sketches))
		for i := range combinations {
			r2 := r.Clone()
			for j := range r2.Sketches {
				// If RequireArchitecture is empty, we repeat the test for all architectures
				// If RequireArchitecture is "*", we execute the test only once with a random architecture
				// If RequireArchitecture is populated, we execute the test only on that architecture
				if r2.Sketches[j].RequireArchitecture == "" {
					r2.Sketches[j].RequireArchitecture = combinations[i][j]
				}
			}
			out = append(out, TestRequirementsMatrix{Original: r, Effective: r2})
		}
	}
	return out
}

func RepeatByFQBNs(tmpl []TestRequirements, fqbns []string) []TestRequirementsMatrix {
	out := make([]TestRequirementsMatrix, 0, len(tmpl)*len(fqbns))
	for _, r := range tmpl {
		combinations := Perm(fqbns, len(r.Sketches))
		for i := range combinations {
			r2 := r.Clone()
			for j := range r2.Sketches {
				// If RequireFQBN is empty, we repeat the test for all FQBNs
				// If RequireFQBN is "*", we execute the test only once with a random FQBN
				// If RequireFQBN is populated, we execute the test only on that FQBN
				if r2.Sketches[j].RequireFQBN == "" {
					r2.Sketches[j].RequireFQBN = combinations[i][j]
				}
			}
			out = append(out, TestRequirementsMatrix{Original: r, Effective: r2})
		}
	}
	return out
}

func uniqRequirements(tr []TestRequirementsMatrix) []TestRequirementsMatrix {
	var out []TestRequirementsMatrix

req:
	for _, r := range tr {
		for _, r2 := range out {
			if r.Effective.Equals(r2.Effective) {
				continue req
			}
		}
		out = append(out, r)
	}

	return out
}

func Perm(set []string, k int) [][]string {
	return perm(set, []string{}, k)
}

func perm(set []string, prefix []string, k int) [][]string {
	if k == 0 {
		// Make a copy of prefix, othwerwise it will always point to the same
		// underlying array, which will be modified in next runs
		prefix = append(prefix[:0:0], prefix...)
		return [][]string{prefix}
	}
	var out [][]string
	for i := 0; i < len(set); i++ {
		o := perm(set, append(prefix, set[i]), k-1)
		out = append(out, o...)
	}
	return out
}
