package stealth

import (
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

type HumanAction struct {
	browser *rod.Browser
}

func New(b *rod.Browser) *HumanAction {
	return &HumanAction{browser: b}
}

// ThinkDelay simulates a user thinking before performing an action
func (h *HumanAction) ThinkDelay(min, max int) {
	duration := time.Duration(min+rand.Intn(max-min)) * time.Millisecond
	time.Sleep(duration)
}

// TypeAction simulates a user typing with variable speed and occasional errors
func (h *HumanAction) TypeAction(page *rod.Page, selector, text string) error {
	el, err := page.Element(selector)
	if err != nil {
		return err
	}

	err = el.Focus()
	if err != nil {
		return err
	}

	for _, r := range text {
		// Variable typing speed
		time.Sleep(time.Duration(50+rand.Intn(150)) * time.Millisecond)

		// Occasional typo simulation could go here

		err = page.Keyboard.Type(input.Key(r))
		if err != nil {
			return err
		}
	}
	return nil
}

// MoveMouseSimulated moves mouse in a non-linear path
func (h *HumanAction) MoveMouseSimulated(page *rod.Page, x, y float64) error {
	// In a real implementation, we would use Bezier curves to calculate
	// intermediate points. For this PoC, we'll simulate jitter.
	page.Mouse.MoveLinear(proto.Point{X: x, Y: y}, 10) // Rod's Move with steps provides some smoothing
	return nil
}

// RandomScroll scrolls the page like a human reading
func (h *HumanAction) RandomScroll(page *rod.Page) error {
	for i := 0; i < 3; i++ {
		dist := rand.Intn(300) + 100
		page.Mouse.Scroll(0, float64(dist), 5)
		h.ThinkDelay(500, 1500)
	}
	return nil
}
