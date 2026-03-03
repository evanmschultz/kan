// Package tillsyndocs embeds top-level repository markdown docs in the binary.
package tillsyndocs

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// MarkdownDocument stores one embedded markdown file payload.
type MarkdownDocument struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
	Markdown string `json:"markdown"`
}

//go:embed *.md
var embeddedMarkdownFS embed.FS

// EmbeddedMarkdownDocuments returns all top-level embedded markdown files in stable filename order.
func EmbeddedMarkdownDocuments() ([]MarkdownDocument, error) {
	entries, err := embeddedMarkdownFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("list embedded markdown docs: %w", err)
	}
	docNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		docNames = append(docNames, name)
	}
	sort.Strings(docNames)

	docs := make([]MarkdownDocument, 0, len(docNames))
	for _, name := range docNames {
		body, readErr := embeddedMarkdownFS.ReadFile(name)
		if readErr != nil {
			return nil, fmt.Errorf("read embedded markdown doc %q: %w", name, readErr)
		}
		docs = append(docs, MarkdownDocument{
			FileName: name,
			Path:     filepath.ToSlash(name),
			Markdown: string(body),
		})
	}
	return docs, nil
}
