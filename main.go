package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

type Job struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	Pipeline     string    `json:"pipeline"`
	Branch       string    `json:"branch"`
	Duration     string    `json:"duration"`
	Started      string    `json:"started"`
	Organization string    `json:"organization"`
	RunID        int64     `json:"run_id"`
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type DashboardStats struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Running int `json:"running"`
	Pending int `json:"pending"`
	Total   int `json:"total"`
}

type RateLimitInfo struct {
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	ResetAt   time.Time `json:"reset_at"`
}

type DashboardResponse struct {
	Stats     DashboardStats `json:"stats"`
	Jobs      []Job          `json:"jobs"`
	RateLimit RateLimitInfo  `json:"rate_limit"`
}

var (
	githubClient *github.Client
	orgNames     []string
)

func init() {
	// Load .env file if it exists
	_ = godotenv.Load()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	orgEnv := os.Getenv("GITHUB_ORG")
	if orgEnv == "" {
		log.Fatal("GITHUB_ORG environment variable is required (can be comma-separated for multiple orgs)")
	}

	// Parse organizations (support comma-separated)
	orgNames = parseOrganizations(orgEnv)
	if len(orgNames) == 0 {
		log.Fatal("At least one organization must be specified in GITHUB_ORG")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient = github.NewClient(tc)
}

func parseOrganizations(orgEnv string) []string {
	orgs := strings.Split(orgEnv, ",")
	var result []string
	for _, org := range orgs {
		org = strings.TrimSpace(org)
		if org != "" {
			result = append(result, org)
		}
	}
	return result
}

func formatDuration(start, end time.Time) string {
	duration := end.Sub(start)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	days := int(diff.Hours() / 24)
	hours := int(diff.Hours())
	minutes := int(diff.Minutes())

	if days > 0 {
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	} else if hours > 0 {
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	} else if minutes > 0 {
		return fmt.Sprintf("%d minute%s ago", minutes, pluralize(minutes))
	}
	return "just now"
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func fetchWorkflowRuns(ctx context.Context, period string) ([]Job, *RateLimitInfo, error) {
	var allJobs []Job
	var rateLimitInfo *RateLimitInfo

	// Determine time range based on period
	now := time.Now()
	var startTime time.Time

	switch period {
	case "today":
		// Untuk "today", gunakan dari jam 1 pagi (01:00:00) hingga jam 11 malam (23:00:00) hari ini
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
		log.Printf("📅 Filter 'today': startTime = %v (now = %v)", startTime, now)
	case "week":
		startTime = now.AddDate(0, 0, -7) // 7 hari yang lalu
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()) // Awal bulan ini
	default:
		startTime = now.AddDate(0, 0, -7) // Default: seminggu terakhir
	}

	log.Printf("📅 Fetching workflow runs for period: %s (since %v)", period, startTime)

	// Loop through all organizations
	for _, orgName := range orgNames {
		log.Printf("📦 Fetching repositories for organization: %s", orgName)

		// Get all repositories in the organization
		repos, resp, err := githubClient.Repositories.ListByOrg(ctx, orgName, &github.RepositoryListByOrgOptions{
			Type: "all",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		})
		if err != nil {
			log.Printf("❌ Error listing repositories for organization %s: %v", orgName, err)
			continue
		}

		log.Printf("✅ Found %d repositories in organization %s", len(repos), orgName)
		if resp != nil {
			log.Printf("   Rate limit: %d/%d remaining (resets at %v)",
				resp.Rate.Remaining, resp.Rate.Limit, resp.Rate.Reset.Time)

			// Store rate limit info (use the latest one)
			rateLimitInfo = &RateLimitInfo{
				Remaining: resp.Rate.Remaining,
				Limit:     resp.Rate.Limit,
				ResetAt:   resp.Rate.Reset.Time,
			}
		}

		// Filter repositories: hanya yang updated dalam periode yang dipilih
		// GitHub web menampilkan "Updated X minutes ago" berdasarkan PushedAt, bukan UpdatedAt
		// Jadi kita perlu cek PushedAt juga, atau gunakan yang lebih baru antara UpdatedAt dan PushedAt
		var filteredRepos []*github.Repository

		for _, repo := range repos {
			var checkTime time.Time
			var hasTime bool

			// Untuk "today", GitHub web biasanya menggunakan PushedAt (waktu commit terakhir)
			// Jadi kita prioritaskan PushedAt, lalu UpdatedAt
			if repo.PushedAt != nil {
				checkTime = repo.PushedAt.Time
				hasTime = true
			} else if repo.UpdatedAt != nil {
				checkTime = repo.UpdatedAt.Time
				hasTime = true
			}

			if hasTime {
				// Convert checkTime ke timezone lokal untuk perbandingan yang benar
				checkTimeLocal := checkTime.In(now.Location())

				// Cek apakah repository di-update dalam periode yang dipilih
				// Gunakan !Before untuk include waktu yang sama dengan startTime
				if !checkTimeLocal.Before(startTime) {
					// Untuk "today", juga cek apakah sebelum jam 11 malam (23:00:00) hari ini
					if period == "today" {
						endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())
						if !checkTimeLocal.After(endTime) {
							filteredRepos = append(filteredRepos, repo)
						}
					} else {
						filteredRepos = append(filteredRepos, repo)
					}
				}
			}
		}

		periodName := map[string]string{
			"today": "today",
			"week":  "this week",
			"month": "this month",
		}[period]
		if periodName == "" {
			periodName = "this week"
		}
		log.Printf("   📅 Filtered: %d repositories updated %s (from %d total)", len(filteredRepos), periodName, len(repos))

		// Fetch workflow runs from repositories updated in selected period
		for i, repo := range filteredRepos {
			log.Printf("   [%d/%d] Fetching workflow runs for repository: %s/%s",
				i+1, len(filteredRepos), orgName, *repo.Name)

			// Get workflow runs (will filter by period in the loop)
			workflowRuns, resp, err := githubClient.Actions.ListRepositoryWorkflowRuns(ctx, orgName, *repo.Name, &github.ListWorkflowRunsOptions{
				ListOptions: github.ListOptions{
					PerPage: 50,
				},
			})
			if err != nil {
				log.Printf("   ❌ Error fetching workflow runs for %s/%s: %v", orgName, *repo.Name, err)
				continue
			}

			if resp != nil {
				log.Printf("   ✅ Found %d workflow runs in %s/%s (Rate limit: %d/%d remaining)",
					len(workflowRuns.WorkflowRuns), orgName, *repo.Name,
					resp.Rate.Remaining, resp.Rate.Limit)

				// Update rate limit info (use the latest one)
				rateLimitInfo = &RateLimitInfo{
					Remaining: resp.Rate.Remaining,
					Limit:     resp.Rate.Limit,
					ResetAt:   resp.Rate.Reset.Time,
				}
			} else {
				log.Printf("   ✅ Found %d workflow runs in %s/%s",
					len(workflowRuns.WorkflowRuns), orgName, *repo.Name)
			}

			for _, run := range workflowRuns.WorkflowRuns {
				// Filter workflow runs berdasarkan waktu untuk semua periode
				var runTime time.Time
				if run.RunStartedAt != nil {
					runTime = run.RunStartedAt.Time
				} else if run.CreatedAt != nil {
					runTime = run.CreatedAt.Time
				} else {
					continue // Skip jika tidak ada timestamp
				}

				// Convert runTime ke timezone lokal untuk perbandingan yang benar
				runTimeLocal := runTime.In(now.Location())

				// Cek apakah dalam periode yang dipilih
				if runTimeLocal.Before(startTime) {
					continue // Skip jika sebelum startTime
				}

				// Untuk "today", juga cek apakah sebelum jam 11 malam (23:00:00) hari ini
				if period == "today" {
					endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())
					if runTimeLocal.After(endTime) {
						continue // Skip jika setelah jam 11 malam hari ini
					}
				}

				status := strings.ToLower(*run.Status)
				conclusion := ""
				if run.Conclusion != nil {
					conclusion = strings.ToLower(*run.Conclusion)
				}

				// Determine job status
				jobStatus := "pending"
				if status == "completed" {
					if conclusion == "success" {
						jobStatus = "success"
					} else if conclusion == "failure" || conclusion == "cancelled" {
						jobStatus = "failed"
					} else {
						jobStatus = "failed"
					}
				} else if status == "in_progress" || status == "queued" {
					jobStatus = "running"
				}

				// Calculate duration
				var duration string
				if run.UpdatedAt != nil && run.RunStartedAt != nil {
					duration = formatDuration(run.RunStartedAt.Time, run.UpdatedAt.Time)
				} else if run.CreatedAt != nil {
					if run.UpdatedAt != nil {
						duration = formatDuration(run.CreatedAt.Time, run.UpdatedAt.Time)
					} else {
						duration = formatDuration(run.CreatedAt.Time, time.Now())
					}
				} else {
					duration = "N/A"
				}

				// Format started time
				var started string
				if run.RunStartedAt != nil {
					started = formatTimeAgo(run.RunStartedAt.Time)
				} else if run.CreatedAt != nil {
					started = formatTimeAgo(run.CreatedAt.Time)
				} else {
					started = "N/A"
				}

				jobName := *run.Name
				if run.RunNumber != nil {
					jobName = fmt.Sprintf("%s #%d", jobName, *run.RunNumber)
				}

				jobID := fmt.Sprintf("JOB-%06d", *run.ID)

				branch := "N/A"
				if run.HeadBranch != nil {
					branch = *run.HeadBranch
				}

				var createdAt time.Time
				if run.CreatedAt != nil {
					createdAt = run.CreatedAt.Time
				} else {
					createdAt = time.Now()
				}

				// Get HTML URL for workflow run detail
				var htmlURL string
				if run.HTMLURL != nil {
					htmlURL = *run.HTMLURL
				} else {
					// Fallback: construct URL manually
					htmlURL = fmt.Sprintf("https://github.com/%s/%s/actions/runs/%d", orgName, *repo.Name, *run.ID)
				}

				job := Job{
					ID:           jobID,
					Name:         jobName,
					Status:       jobStatus,
					Pipeline:     *repo.Name, // Repository name instead of workflow name
					Branch:       branch,
					Duration:     duration,
					Started:      started,
					Organization: orgName,
					RunID:        *run.ID,
					HTMLURL:      htmlURL,
					CreatedAt:    createdAt,
				}

				allJobs = append(allJobs, job)
			}
		}

		log.Printf("✅ Completed fetching for organization %s. Total jobs collected: %d",
			orgName, len(allJobs))
	}

	log.Printf("📊 Total jobs collected from all organizations: %d", len(allJobs))

	// Sort jobs by CreatedAt (newest first)
	sort.Slice(allJobs, func(i, j int) bool {
		return allJobs[i].CreatedAt.After(allJobs[j].CreatedAt)
	})

	// Return default rate limit if not set
	if rateLimitInfo == nil {
		rateLimitInfo = &RateLimitInfo{
			Remaining: 5000,
			Limit:     5000,
			ResetAt:   time.Now().Add(1 * time.Hour),
		}
	}

	return allJobs, rateLimitInfo, nil
}

func calculateStats(jobs []Job) DashboardStats {
	stats := DashboardStats{
		Total: len(jobs),
	}

	for _, job := range jobs {
		switch job.Status {
		case "success":
			stats.Success++
		case "failed":
			stats.Failed++
		case "running":
			stats.Running++
		case "pending":
			stats.Pending++
		}
	}

	return stats
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("🌐 Dashboard API request from %s", r.RemoteAddr)
	ctx := context.Background()

	// Get period parameter from query string (default: week)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week" // Default: seminggu terakhir
	}

	// Validate period
	if period != "today" && period != "week" && period != "month" {
		period = "week"
	}

	startTime := time.Now()
	jobs, rateLimit, err := fetchWorkflowRuns(ctx, period)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ Error fetching workflow runs: %v (took %v)", err, duration)
		http.Error(w, fmt.Sprintf("Error fetching workflow runs: %v", err), http.StatusInternalServerError)
		return
	}

	stats := calculateStats(jobs)
	log.Printf("📈 Dashboard stats: Success=%d, Failed=%d, Running=%d, Pending=%d, Total=%d (took %v)",
		stats.Success, stats.Failed, stats.Running, stats.Pending, stats.Total, duration)

	// Set default rate limit if nil
	if rateLimit == nil {
		rateLimit = &RateLimitInfo{
			Remaining: 5000,
			Limit:     5000,
			ResetAt:   time.Now().Add(1 * time.Hour),
		}
	}

	response := DashboardResponse{
		Stats:     stats,
		Jobs:      jobs,
		RateLimit: *rateLimit,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/dashboard", dashboardHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
