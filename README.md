# Crawl

Crawl is a simple web crawler, written as a package to use as you want but also as an application in [`./cmd/cli.go`](./cmd/cli.go) the idea
was to create this as some sort of cli application, but I did not have time.

## Notes for reviewer

- I would suggest starting from [Design Document](./DESIGN.md) to get an understanding of my thought process.
- Then open up [./cmd/cli.go](./cmd/cli.go) to see the entry point.
- Follow the code in [./orchestrator.go](./orchestrator.go), this is where the important code is.