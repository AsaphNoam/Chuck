package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
)

// runResumeByID POSTs to /api/sessions/{id}/resume and prints the result.
func runResumeByID(agentID string) int {
	cfgStore, cerr := config.New()
	if cerr != nil {
		fmt.Printf("config: %v\n", cerr)
		return 1
	}
	port := dashboardPort(cfgStore)

	resp, body, err := postResume(port, agentID, nil)
	if err != nil {
		fmt.Printf("could not reach dashboard on 127.0.0.1:%d — run `chuck dashboard start` first (%v)\n", port, err)
		return 1
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("resume failed (%d): %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		return 1
	}

	var out struct {
		Agent struct {
			AgentID string `json:"agent_id"`
			Name    string `json:"name"`
			Role    string `json:"role"`
			Project string `json:"project"`
		} `json:"agent"`
	}
	_ = json.Unmarshal(body, &out)
	fmt.Printf("resumed %s (%s) %s@%s\n", out.Agent.Name, out.Agent.AgentID, out.Agent.Role, out.Agent.Project)
	return 0
}

// postResume POSTs to /api/sessions/{id}/resume with an optional body.
func postResume(port int, agentID string, body map[string]string) (*http.Response, []byte, error) {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/sessions/%s/resume", port, agentID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data, nil
}

// archiveResult is one session from a project-scoped archive read.
type archiveResult struct {
	AgentID string `json:"agent_id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Project string `json:"project"`
	Active  bool   `json:"active"`
}

const (
	archiveLookupPageSize = 200
	archiveLookupMaxPages = 10
)

// listInactiveSessions queries the archive endpoint for inactive sessions
// matching the given role and project. When name is non-empty it also filters
// by name, so a bare `role@project --name X` only auto-resumes the session
// actually named X rather than any inactive role@project session.
func listInactiveSessions(port int, role, project, name string) ([]archiveResult, error) {
	// The global archive is grouped by project. Its existing project detail
	// route supplies the flat agent rows the legacy matching rule consumes.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := &operatorClient{HTTP: &http.Client{}, BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port)}
	var matches []archiveResult
	var total *int
	for page := 0; page < archiveLookupMaxPages; page++ {
		offset := page * archiveLookupPageSize
		path := fmt.Sprintf("/api/archive/projects/%s?active=false&limit=%d", url.PathEscape(project), archiveLookupPageSize)
		if offset > 0 {
			path += fmt.Sprintf("&offset=%d", offset)
		}
		data, err := client.Request(ctx, http.MethodGet, path, nil, nil)
		if err != nil {
			return nil, err
		}

		var ar struct {
			Total   *int            `json:"total"`
			Results []archiveResult `json:"results"`
		}
		if err := json.Unmarshal(data, &ar); err != nil {
			return nil, err
		}
		if ar.Total == nil {
			return nil, fmt.Errorf("archive lookup response omitted total")
		}
		if *ar.Total < 0 || *ar.Total < offset+len(ar.Results) || len(ar.Results) > archiveLookupPageSize {
			return nil, fmt.Errorf("archive lookup returned an invalid page at offset %d", offset)
		}
		if total != nil && *ar.Total != *total {
			return nil, fmt.Errorf("archive lookup total changed during pagination")
		}
		if page == 0 && *ar.Total > archiveLookupPageSize*archiveLookupMaxPages {
			return nil, fmt.Errorf("archive lookup exceeds the bounded search of %d sessions", archiveLookupPageSize*archiveLookupMaxPages)
		}
		if total == nil {
			total = ar.Total
		}
		if len(ar.Results) == 0 {
			if offset < *ar.Total {
				return nil, fmt.Errorf("archive lookup returned an incomplete page at offset %d", offset)
			}
			return matches, nil
		}
		if len(ar.Results) < archiveLookupPageSize && offset+len(ar.Results) < *ar.Total {
			return nil, fmt.Errorf("archive lookup returned an incomplete page at offset %d", offset)
		}
		for _, r := range ar.Results {
			if !r.Active && r.Role == role && r.Project == project && (name == "" || r.Name == name) {
				matches = append(matches, r)
			}
		}
		if len(matches) > 1 {
			return matches, nil
		}
		if offset+len(ar.Results) >= *ar.Total {
			return matches, nil
		}
	}
	return nil, fmt.Errorf("archive lookup exceeded the bounded search of %d sessions", archiveLookupPageSize*archiveLookupMaxPages)
}
