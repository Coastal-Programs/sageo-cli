package crawl

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/jakeschepis/sageo-cli/internal/provider"
)

const (
	defaultDepth    = 2
	defaultMaxPages = 50
	concurrency     = 5
)

type crawler struct {
	fetcher provider.Fetcher
}

type crawlItem struct {
	url   string
	depth int
}

func (c *crawler) Run(ctx context.Context, req Request) (Result, error) {
	depth := req.Depth
	if depth <= 0 {
		depth = defaultDepth
	}
	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = defaultMaxPages
	}

	parsedBase, err := url.Parse(req.TargetURL)
	if err != nil {
		return Result{}, err
	}

	result := Result{TargetURL: req.TargetURL}

	var (
		mu      sync.Mutex
		visited = map[string]bool{}
		wg      sync.WaitGroup
		sem     = make(chan struct{}, concurrency)
		queue   = make(chan crawlItem, maxPages*10)
	)

	// Normalize and mark the start URL as visited
	startURL := normalizeURL(parsedBase)
	visited[startURL] = true
	queue <- crawlItem{url: startURL, depth: 0}

	// Track active work to know when to close the queue
	active := 1

	for item := range queue {
		// Abort on context cancellation
		if ctx.Err() != nil {
			mu.Lock()
			active--
			if active == 0 {
				close(queue)
			}
			mu.Unlock()
			continue
		}

		// Check page limit
		mu.Lock()
		pageCount := len(result.Pages)
		mu.Unlock()
		if pageCount >= maxPages {
			mu.Lock()
			active--
			if active == 0 {
				close(queue)
			}
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(item crawlItem) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			// Abort on context cancellation
			if ctx.Err() != nil {
				mu.Lock()
				active--
				if active == 0 {
					close(queue)
				}
				mu.Unlock()
				return
			}

			// Check limit before fetching
			mu.Lock()
			if len(result.Pages) >= maxPages {
				active--
				if active == 0 {
					close(queue)
				}
				mu.Unlock()
				return
			}
			mu.Unlock()

			fetchResult, fetchErr := c.fetcher.Fetch(ctx, item.url)

			mu.Lock()
			if fetchErr != nil {
				result.Errors = append(result.Errors, CrawlError{
					URL:     item.url,
					Message: fetchErr.Error(),
				})
				active--
				if active == 0 {
					close(queue)
				}
				mu.Unlock()
				return
			}

			// Double-check limit after fetch (another goroutine may have filled it)
			if len(result.Pages) >= maxPages {
				active--
				if active == 0 {
					close(queue)
				}
				mu.Unlock()
				return
			}

			page := extractPageData(item.url, fetchResult.StatusCode, fetchResult.Body)
			result.Pages = append(result.Pages, page)
			currentCount := len(result.Pages)
			mu.Unlock()

			// Enqueue discovered internal links if within depth limit
			if item.depth < depth && currentCount < maxPages {
				for _, link := range page.Links {
					if !link.Internal {
						continue
					}
					parsed, parseErr := url.Parse(link.Href)
					if parseErr != nil {
						continue
					}
					normalized := normalizeURL(parsed)
					if !isSameDomain(parsed, parsedBase) {
						continue
					}
					// Skip non-HTTP schemes, fragments, etc.
					if parsed.Scheme != "http" && parsed.Scheme != "https" {
						continue
					}

					mu.Lock()
					if !visited[normalized] && len(result.Pages) < maxPages {
						visited[normalized] = true
						active++
						mu.Unlock()
						select {
						case queue <- crawlItem{url: normalized, depth: item.depth + 1}:
						default:
						}
					} else {
						mu.Unlock()
					}
				}
			}

			mu.Lock()
			active--
			if active == 0 {
				close(queue)
			}
			mu.Unlock()
		}(item)
	}

	wg.Wait()

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}

func normalizeURL(u *url.URL) string {
	// Strip fragment, ensure trailing slash consistency
	u.Fragment = ""
	path := u.Path
	if path == "" {
		path = "/"
	}
	return u.Scheme + "://" + u.Host + path
	// Note: preserves query params in path; for crawling we keep it simple
}

func isSameDomain(link, base *url.URL) bool {
	return strings.EqualFold(link.Host, base.Host)
}
