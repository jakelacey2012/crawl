package main

import (
	"crawl"
	"flag"
	"fmt"
	"github.com/charmbracelet/log"
	"net/http"
	"time"
)

func main() {

	orch := crawl.Orchestrator{}

	var maxRequests = flag.Uint64("max_requests", 1_000_000, "max requests to make")
	var politenessDelay = flag.Uint64("politeness_delay", 100, "delay between requests")
	var timeout = flag.Uint64("timeout", 10000000, "timeout for the crawler")
	var startUrl = flag.String("start_url", "", "start url")

	log.SetLevel(log.InfoLevel)
	flag.Parse()

	if *startUrl == "" {
		log.Fatal("start_url is required")
	}

	orch.Init(&crawl.OrchestratorConfig{
		MaxRequests:     *maxRequests,
		PolitenessDelay: *politenessDelay,
		Timeout:         time.Duration(*timeout),
	})

	orch.OnVisited(func(resp *http.Response) {
		fmt.Printf("%v \n", resp.Request.URL.String())
	})

	orch.Crawl(*startUrl)
}
