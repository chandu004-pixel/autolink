package search

import (
	"strconv"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/user/autolink/internal/logging"
	"github.com/user/autolink/internal/stealth"
)

type ProfileResult struct {
	ID        int
	Name      string
	Title     string
	Company   string
	Connected bool
}

type Service struct {
	Stealth *stealth.HumanAction
}

func New(s *stealth.HumanAction) *Service {
	return &Service{Stealth: s}
}

func (s *Service) ExecuteSearch(page *rod.Page, query string) ([]ProfileResult, error) {
	logging.Logger.Infof("Searching for: %s", query)

	err := page.Navigate("http://localhost:8080/search")
	if err != nil {
		return nil, err
	}

	err = s.Stealth.TypeAction(page, "#search-input", query)
	if err != nil {
		return nil, err
	}

	// Press Enter to submit search
	err = page.Keyboard.Press(input.Enter)
	if err != nil {
		return nil, err
	}

	s.Stealth.ThinkDelay(1000, 2000)

	// Parse results
	items, err := page.Elements(".result-item")
	if err != nil {
		return nil, err
	}

	results := []ProfileResult{}
	for _, item := range items {
		idStr, _ := item.Attribute("data-id")
		id := 0
		if idStr != nil {
			id, _ = strconv.Atoi(*idStr)
		}

		// Updated selector for the name link in the new UI
		nameEl, err := item.Element("h3 a")
		if err != nil {
			continue
		}
		name := nameEl.MustText()

		// Robust connectivity check matching the new 'NETWORK SYNCED' terminology
		itemText := item.MustText()
		connected := strings.Contains(itemText, "NETWORK SYNCED") || strings.Contains(itemText, "Connected")

		// Extract title and company with better targeting
		title := ""
		company := ""                     // New field extraction
		titleEl, err := item.Element("p") // The title and company are in a <p> tag
		if err == nil {
			fullText := titleEl.MustText()
			// Text is like "Title at Company"
			parts := strings.Split(fullText, " at ")
			if len(parts) > 0 {
				title = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				company = strings.TrimSpace(parts[1])
			}
		}

		results = append(results, ProfileResult{
			ID:        id,
			Name:      name,
			Title:     title,
			Company:   company,
			Connected: connected,
		})
	}

	logging.Logger.Infof("Found %d results", len(results))
	return results, nil
}
