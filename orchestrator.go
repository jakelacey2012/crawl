package crawl

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type VType uint8

const (
	TO_VISIT VType = 0
	VISITED  VType = 1
)

// OrchestratorStore is to store data during the crawling process(es).
type OrchestratorStore struct {
	// Keep track of the visited URLs.
	Visited map[string]VType
	// Lock to ensure that the store is thread-safe.
	mu sync.Mutex
}

// OrchestratorConfig is the configuration struct that will be used to control the crawling process(es).
type OrchestratorConfig struct {
	// MaxRequests is the maximum number of requests to make.
	MaxRequests uint64
	// PolitenessDelay is the delay between requests.
	PolitenessDelay uint64
	// Timeout is the maximum time to wait for the crawling process(es) to finish.
	Timeout time.Duration
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Orchestrator is the main struct that will be used to control the crawling process(es).
type Orchestrator struct {
	// Context is the context that will be used to control the crawling process(es).
	Context context.Context
	// Config is the configuration that will be used to control the crawling process(es).
	Config OrchestratorConfig
	// Store is the storage that will be used to store data during the crawling process(es).
	Store OrchestratorStore
	// VisitedCallback is the callback that will be called when a URL is visited.
	VisitedCallback []func(resp *http.Response)
	// Queue is the queue that will be used to store URLs to visit.
	Queue chan string
	// processedOps is the atomic counter that will be used to keep track of the number of operations.
	processedOps atomic.Uint64
	// sentOps is the atomic counter that will be used to keep track of the number of operations.
	sendingOps atomic.Uint64
	// Client is the HTTP client that will be used to make requests.
	Client HttpClient
}

// Init initializes the Orchestrator struct with default values.
func (o *Orchestrator) Init(config *OrchestratorConfig) {
	o.Context = context.Background()
	o.Config = *config
	o.Client = &http.Client{
		Timeout: 2 * time.Second,
	}
	o.Queue = make(chan string, o.Config.MaxRequests)
	o.Store = OrchestratorStore{
		Visited: make(map[string]VType),
	}
}

// OnVisited registers a callback that will be called when a URL is visited.
func (o *Orchestrator) OnVisited(callback func(resp *http.Response)) {
	o.VisitedCallback = append(o.VisitedCallback, callback)
}

func (o *Orchestrator) visited(resp *http.Response) {
	for _, callback := range o.VisitedCallback {
		callback(resp)
	}
}

type Request struct {
	ID  uint64
	Url string
}

// Performs a HTTP request to the given URL, and passes the result to a given channel.
func (o *Orchestrator) requester(id int, requests <-chan Request, results chan<- io.ReadCloser) {
	for request := range requests {
		if o.processedOps.Load() > o.Config.MaxRequests {
			log.Infof("Max Requests Limit Reached Exiting")
			o.processedOps.Add(1)
			continue
		}

		// Not sure if I need this lock but maybe if two requesters are processing the same
		o.Store.mu.Lock()
		if v, ok := o.Store.Visited[request.Url]; ok && v == VISITED {
			o.Store.mu.Unlock()
			o.processedOps.Add(1)
			continue
		}

		req, err := http.NewRequest("GET", request.Url, nil)
		if err != nil {
			o.Store.mu.Unlock()
			log.Infof("Error creating request: %v", err)
			o.processedOps.Add(1)
			continue
		}

		resp, err := o.Client.Do(req)
		if err != nil {
			o.Store.mu.Unlock()
			log.Infof("Error creating request: %v", err)
			o.processedOps.Add(1)
			continue
		}

		o.Store.Visited[request.Url] = VISITED
		o.Store.mu.Unlock()
		o.visited(resp)

		o.processedOps.Add(1)

		contentType := resp.Header.Get("Content-type")
		if strings.HasPrefix(contentType, "text/html") == false {
			log.Infof("Is not HTML content type: %v", contentType)
			continue
		}

		results <- resp.Body
	}
}

// Parses the links from the body of a HTTP response, and passes the result to a given channel.
func (o *Orchestrator) linkerParser(id int, bodies <-chan io.ReadCloser, links chan<- string) {
	for body := range bodies {

		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			log.Errorf("Error reading body: %v", err)
			continue
		}

		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			link, _ := s.Attr("href")
			links <- link
		})

		if err := body.Close(); err != nil {
			log.Errorf("Error closing body: %v", err)
			continue
		}
	}
}

func (o *Orchestrator) processWorkingLinks(id int, unvalidatedLinks <-chan string, startingUrl *url.URL) {
	for unvalidatedLink := range unvalidatedLinks {
		link, err := url.Parse(unvalidatedLink)
		if err != nil {
			log.Debugf("Error parsing URL: %v", err)
			continue
		}

		if link.IsAbs() == false {
			link = startingUrl.ResolveReference(link)
		}

		startingDomain := startingUrl.Hostname()
		linkDomain := link.Hostname()

		if startingDomain != linkDomain {
			continue
		}

		o.Store.mu.Lock()
		joined, _ := url.JoinPath(link.String(), "/")
		if _, ok := o.Store.Visited[joined]; ok {
			log.Debugf("  Visited already skipping link: %v", link)
			o.Store.mu.Unlock()
			continue
		}

		o.Store.Visited[joined] = TO_VISIT
		o.Store.mu.Unlock()

		if o.processedOps.Load() > o.Config.MaxRequests {
			log.Debugf("Max Requests Limit Reached Exiting")
			continue
		}

		// Here we make sure we don't fill up the buffer and cause a deadlock, this does have the downside that we will
		// "drop" those links and not visit them. This is a blind spot for me in go, so it would be interesting to see if
		// there is a way to deal with this better :)
		if len(o.Queue) <= int(o.Config.MaxRequests-(o.Config.MaxRequests/8)) {
			o.Queue <- joined
		}
	}
}

// Crawl Crawls the given URL.
func (o *Orchestrator) Crawl(startUrlString string) {

	str, err := url.JoinPath(startUrlString, "/")
	if err != nil {
		log.Fatalf("Error parsing URL: %v", err)
	}

	startUrl, err := url.Parse(str)
	if err != nil {
		log.Fatalf("Error parsing URL: %v", err)
	}

	ticker := time.NewTicker(time.Duration(o.Config.PolitenessDelay) * time.Millisecond)
	sleep, cancel := context.WithTimeout(o.Context, o.Config.Timeout*time.Millisecond)
	defer cancel()

	jobs := make(chan Request)
	payloads := make(chan io.ReadCloser)
	workingLinks := make(chan string)

	for w := 1; w <= 100; w++ {
		go o.requester(w, jobs, payloads)
		go o.linkerParser(w, payloads, workingLinks)
		go o.processWorkingLinks(w, workingLinks, startUrl)
	}

	go func() {
		for {
			select {
			case <-o.Context.Done():
				return
			case _ = <-ticker.C:
				log.Debug("Tick")

				// Let's add a bit of observability so that we can see what happens
				// I would replace this with a counter which is emitted in production.
				log.Debugf("TQueue %v", len(o.Queue))
				log.Debugf("TSend %v", o.sendingOps.Load())
				log.Debugf("TProc %v", o.processedOps.Load())
				log.Debugf("TMax %v", o.Config.MaxRequests)

				// Let's check if we should be exiting the ticker
				if o.processedOps.Load() >= o.Config.MaxRequests {
					log.Info("Max requests reached, stopping...")
					cancel()
					return
				}
				if len(o.Queue) == 0 && o.sendingOps.Load() == o.processedOps.Load() {
					log.Info("  Queue is empty, stopping...")
					cancel()
					return
				}

				if len(o.Queue) > 0 {
					link := <-o.Queue
					o.sendingOps.Add(1)
					jobs <- Request{ID: o.sendingOps.Load(), Url: link}
				}

				log.Debug("Tock")
			}
		}
	}()

	log.Info("Initiating Crawling")

	o.Queue <- startUrl.String()

	<-sleep.Done()
	ticker.Stop()
	close(jobs)
	close(payloads)
	close(workingLinks)

	log.Info("Finished Crawling")
}
