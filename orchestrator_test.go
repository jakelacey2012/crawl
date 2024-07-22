package crawl

import (
	"bytes"
	"fmt"
	"golang.org/x/exp/slices"
	"io"
	"net/http"
	"testing"
)

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestOrchestrator_Crawl2(t *testing.T) {
	orch := Orchestrator{}

	orch.Init(&OrchestratorConfig{
		MaxRequests:     1000,
		PolitenessDelay: 50,
		Timeout:         5000,
	})

	orch.OnVisited(func(resp *http.Response) {
		fmt.Printf("Visited: (%v) %v \n", resp.StatusCode, resp.Request.URL.String())
	})

	orch.Crawl("https://monzo.com")
}

func TestOrchestrator_Init(t *testing.T) {
	orch := Orchestrator{}
	orch.Init(&OrchestratorConfig{
		MaxRequests:     1,
		PolitenessDelay: 1,
	})

	if orch.Config.MaxRequests != 1 {
		t.Errorf("Expected MaxRequests to be 1, got %d", orch.Config.MaxRequests)
	}

	if orch.Config.PolitenessDelay != 1 {
		t.Errorf("Expected PolitenessDelay to be 1, got %d", orch.Config.PolitenessDelay)
	}

	if orch.Context == nil {
		t.Error("Expected Context to be initialized")
	}
}

func TestOrchestrator_OnVisited(t *testing.T) {
	orch := Orchestrator{}
	orch.OnVisited(func(resp *http.Response) {})

	if len(orch.VisitedCallback) != 1 {
		t.Errorf("Expected VisitedCallback to have 1 element, got %d", len(orch.VisitedCallback))
	}
}

func TestOrchestrator_Crawl(t *testing.T) {
	htmlContentPage1 := `
		<html>
			<body>
				<a href="https://test.com/link/1">Link 1</a>
				<a href="https://test.com/link/2">Link 2</a>
			</body>
		</html>	
	`

	htmlContentForLink1 := `
		<html>
			<body>
				<a href="https://test.com/link/3">Link 3</a>
				<a href="https://test.com/link/4">Link 4</a>
			</body>
		</html>	
	`

	htmlContentForLink2 := `
		<html>
			<body>
				<a href="https://test.com/link/5">Link 5</a>
				<a href="https://test.com/link/6">Link 6</a>
			</body>
		</html>	
	`

	htmlContentForLink3 := `
		<html>
			<body>
				<!-- this contains duplicates -->
				<a href="https://test.com/link/1">Link 1</a>
				<a href="https://test.com/link/2">Link 2</a>
				<a href="https://test.com/link/3">Link 3</a>
				<a href="https://test.com/link/4">Link 4</a>
				<a href="https://test.com/link/5">Link 5</a>
				<a href="https://test.com/link/6">Link 6</a>
			</body>
		</html>	
	`

	htmlContentForLink4 := `
		<html>
			<body>
				<a href="https://test.com/link/100">Link 100</a>
			</body>
		</html>	
	`

	orch := Orchestrator{}

	orch.Init(&OrchestratorConfig{
		MaxRequests:     100,
		PolitenessDelay: 400,
		Timeout:         4000,
	})

	orch.Client = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://test.com/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentPage1)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/link/1/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentForLink1)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/link/2/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentForLink2)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/link/3/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentForLink3)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/link/4/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentForLink4)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			default:
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			}
		},
	}

	var responses []string

	orch.OnVisited(func(resp *http.Response) {
		responses = append(responses, resp.Request.URL.String())
	})

	orch.Crawl("https://test.com")

	if len(responses) != 8 {
		t.Errorf("Should have visited 8 urls, only visited %d.", len(responses))
	}

	linksToVerify := []string{
		"https://test.com/",
		"https://test.com/link/1/",
		"https://test.com/link/2/",
		"https://test.com/link/3/",
		"https://test.com/link/4/",
		"https://test.com/link/5/",
		"https://test.com/link/6/",
	}

	for _, link := range linksToVerify {
		if !slices.Contains(responses, link) {
			t.Errorf("responses should contain link %v", link)
		}
	}
}

func TestOrchestrator_CrawlWithDifferentStatusCodes(t *testing.T) {
	orch := Orchestrator{}

	htmlContentPage1 := `
			<html>
				<body>
					<a href="https://test.com/status/200">Status 200</a>
					<a href="https://test.com/status/204">Status 204</a>
					<a href="https://test.com/status/404">Status 404</a>
					<a href="https://test.com/status/401">Status 401</a>
					<a href="https://test.com/status/400">Status 400</a>
					<a href="https://test.com/status/500">Status 500</a>
				</body>
			</html>	
		`

	htmlContentFromStatus200Path := `
			<html>
				<body>
					<a href="https://test.com/foo/bar/200">Foo Bar One</a>
				</body>
			</html>	
		`

	htmlContentFromStatus204Path := `
			<html>
				<body>
					<a href="https://test.com/foo/bar/204">Foo Bar Two o Four</a>
				</body>
			</html>	
		`

	htmlContentFromStatus404Path := `
			<html>
				<body>
					<a href="https://test.com/foo/bar/404">Foo Bar Four o Four</a>
				</body>
			</html>	
		`

	htmlContentFromStatus401Path := `
			<html>
				<body>
					<a href="https://test.com/foo/bar/401">Foo Bar Four o One</a>
				</body>
			</html>	
		`

	htmlContentFromStatus400Path := `
		<html>
			<body>
				<a href="https://test.com/foo/bar/400">Foo Bar Four o o</a>
			</body>
		</html>	
	`

	orch.Init(&OrchestratorConfig{
		MaxRequests:     100,
		PolitenessDelay: 400,
		Timeout:         10000,
	})

	orch.Client = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://test.com/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentPage1)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/status/200/":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentFromStatus200Path)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/status/204/":
				return &http.Response{
					StatusCode: 204,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentFromStatus204Path)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/status/404/":
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentFromStatus404Path)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/status/401/":
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentFromStatus401Path)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			case "https://test.com/status/400/":
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewBufferString(htmlContentFromStatus400Path)),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			default:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     map[string][]string{"Content-Type": {"text/html"}},
					Request:    req,
				}, nil
			}
		},
	}

	var responses []string

	orch.OnVisited(func(resp *http.Response) {
		responses = append(responses, resp.Request.URL.String())
	})

	orch.Crawl("https://test.com")

	if len(responses) != 12 {
		t.Errorf("Should have visited 12 urls, only visited %d.", len(responses))
	}

	linksToVerify := []string{
		"https://test.com/",
		"https://test.com/status/200/",
		"https://test.com/status/204/",
		"https://test.com/status/404/",
		"https://test.com/status/401/",
		"https://test.com/status/400/",
		"https://test.com/status/500/",
		"https://test.com/foo/bar/200/",
		"https://test.com/foo/bar/204/",
		"https://test.com/foo/bar/404/",
		"https://test.com/foo/bar/401/",
		"https://test.com/foo/bar/400/",
	}

	for _, link := range linksToVerify {
		if !slices.Contains(responses, link) {
			t.Errorf("responses should contain link %v", link)
		}
	}
}
