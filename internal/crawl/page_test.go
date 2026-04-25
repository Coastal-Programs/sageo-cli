package crawl

import "testing"

func TestExtractPageDataNormalPage(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta name="description" content="A test description">
	<link rel="canonical" href="https://example.com/test">
</head>
<body>
	<h1>Main Heading</h1>
	<h2>Sub Heading</h2>
	<a href="/about">About</a>
	<a href="https://external.com">External</a>
	<img src="/logo.png" alt="Logo">
	<img src="/photo.jpg">
</body>
</html>`)

	result := extractPageData("https://example.com/test", 200, html, nil, 0)

	if result.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", result.Title)
	}
	if result.MetaDescription != "A test description" {
		t.Errorf("expected meta description 'A test description', got %q", result.MetaDescription)
	}
	if result.Canonical != "https://example.com/test" {
		t.Errorf("expected canonical 'https://example.com/test', got %q", result.Canonical)
	}
	if len(result.Headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(result.Headings))
	}
	if result.Headings[0].Level != 1 || result.Headings[0].Text != "Main Heading" {
		t.Errorf("unexpected h1: %+v", result.Headings[0])
	}
	if result.Headings[1].Level != 2 || result.Headings[1].Text != "Sub Heading" {
		t.Errorf("unexpected h2: %+v", result.Headings[1])
	}
	if len(result.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(result.Links))
	}
	if !result.Links[0].Internal {
		t.Error("expected first link to be internal")
	}
	if result.Links[1].Internal {
		t.Error("expected second link to be external")
	}
	if len(result.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(result.Images))
	}
	if result.Images[0].Alt != "Logo" {
		t.Errorf("expected alt 'Logo', got %q", result.Images[0].Alt)
	}
	if result.Images[1].Alt != "" {
		t.Errorf("expected empty alt, got %q", result.Images[1].Alt)
	}
}

func TestExtractPageDataMissingElements(t *testing.T) {
	html := []byte(`<html><body><p>No SEO elements</p></body></html>`)
	result := extractPageData("https://example.com", 200, html, nil, 0)

	if result.Title != "" {
		t.Errorf("expected empty title, got %q", result.Title)
	}
	if result.MetaDescription != "" {
		t.Errorf("expected empty meta description, got %q", result.MetaDescription)
	}
	if result.Canonical != "" {
		t.Errorf("expected empty canonical, got %q", result.Canonical)
	}
	if len(result.Headings) != 0 {
		t.Errorf("expected no headings, got %d", len(result.Headings))
	}
}

func TestExtractPageDataMalformedHTML(t *testing.T) {
	// Malformed: missing closing tags, nested improperly
	html := []byte(`<html><head><title>Broken</title></head><body><h1>OK<img src="/x"><p>unclosed`)
	result := extractPageData("https://example.com", 200, html, nil, 0)

	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
	if result.Title != "Broken" {
		t.Errorf("expected title 'Broken', got %q", result.Title)
	}
	// The parser should still extract the h1
	found := false
	for _, h := range result.Headings {
		if h.Level == 1 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find h1 in malformed HTML")
	}
}

func TestExtractPageDataMetaAuthor(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head>
	<title>Author Test</title>
	<meta name="author" content="Jane Doe">
</head>
<body><h1>x</h1></body>
</html>`)
	result := extractPageData("https://example.com", 200, html, nil, 0)
	if result.MetaAuthor != "Jane Doe" {
		t.Errorf("expected MetaAuthor 'Jane Doe', got %q", result.MetaAuthor)
	}
}

func TestExtractPageDataSchemasBody(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head>
	<title>Schema Test</title>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "Article",
		"author": {"@type": "Person", "name": "Jane Doe"}
	}
	</script>
</head>
<body><h1>x</h1></body>
</html>`)
	result := extractPageData("https://example.com", 200, html, nil, 0)
	if len(result.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(result.Schemas))
	}
	author, ok := result.Schemas[0]["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected author to be a map, got %T", result.Schemas[0]["author"])
	}
	if author["@type"] != "Person" {
		t.Errorf("expected author @type 'Person', got %v", author["@type"])
	}
	// Confirm SchemaTypes still works (backwards compat)
	if len(result.SchemaTypes) != 1 || result.SchemaTypes[0] != "Article" {
		t.Errorf("expected SchemaTypes [Article], got %v", result.SchemaTypes)
	}
}
