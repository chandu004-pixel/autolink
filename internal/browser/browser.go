package browser

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type Client struct {
	Browser *rod.Browser
}

func New(headless bool) (*Client, error) {
	l := launcher.New().
		Headless(headless).
		UserDataDir("./.browser_data").
		Set("no-sandbox").
		Set("disable-setuid-sandbox").
		Set("disable-blink-features", "AutomationControlled")

	// If running in Docker, Chromium is usually at /usr/bin/chromium
	if path, exists := launcher.LookPath(); exists {
		l.Bin(path)
	}

	url, err := l.Launch()
	if err != nil {
		return nil, err
	}

	b := rod.New().ControlURL(url).MustConnect()

	return &Client{Browser: b}, nil
}

func (c *Client) NewStealthPage(url string) (*rod.Page, error) {
	page := c.Browser.MustPage(url)
	return page, nil
}

func (c *Client) Close() error {
	return c.Browser.Close()
}
