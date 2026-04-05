package crawl

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// extractPageData parses HTML and extracts SEO-relevant data from a page.
func extractPageData(pageURL string, statusCode int, body []byte) PageResult {
	result := PageResult{
		URL:        pageURL,
		StatusCode: statusCode,
	}

	parsedBase, _ := url.Parse(pageURL)

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return result
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if text := textContent(n); text != "" {
					result.Title = text
				}
			case "meta":
				handleMeta(n, &result)
			case "link":
				handleLink(n, &result)
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := int(n.Data[1] - '0')
				result.Headings = append(result.Headings, Heading{
					Level: level,
					Text:  textContent(n),
				})
			case "a":
				handleAnchor(n, parsedBase, &result)
			case "img":
				handleImg(n, &result)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return result
}

func handleMeta(n *html.Node, result *PageResult) {
	var name, content string
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "name":
			name = strings.ToLower(a.Val)
		case "content":
			content = a.Val
		}
	}
	if name == "description" {
		result.MetaDescription = content
	}
}

func handleLink(n *html.Node, result *PageResult) {
	var rel, href string
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "rel":
			rel = strings.ToLower(a.Val)
		case "href":
			href = a.Val
		}
	}
	if rel == "canonical" && href != "" {
		result.Canonical = href
	}
}

func handleAnchor(n *html.Node, base *url.URL, result *PageResult) {
	var href string
	for _, a := range n.Attr {
		if a.Key == "href" {
			href = a.Val
			break
		}
	}
	if href == "" {
		return
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return
	}

	resolved := base.ResolveReference(parsed)
	internal := resolved.Host == base.Host

	result.Links = append(result.Links, Link{
		Href:     resolved.String(),
		Text:     textContent(n),
		Internal: internal,
	})
}

func handleImg(n *html.Node, result *PageResult) {
	var src, alt string
	for _, a := range n.Attr {
		switch a.Key {
		case "src":
			src = a.Val
		case "alt":
			alt = a.Val
		}
	}
	result.Images = append(result.Images, Image{
		Src: src,
		Alt: alt,
	})
}

// textContent returns the concatenated text content of a node and its children.
func textContent(n *html.Node) string {
	var sb strings.Builder
	var collect func(*html.Node)
	collect = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(n)
	return strings.TrimSpace(sb.String())
}
