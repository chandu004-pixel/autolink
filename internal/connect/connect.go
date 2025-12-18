package connect

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/user/autolink/internal/logging"
	"github.com/user/autolink/internal/retry"
	"github.com/user/autolink/internal/stealth"
	"github.com/user/autolink/internal/storage"
)

type Service struct {
	Stealth *stealth.HumanAction
	Store   *storage.Store
}

func New(s *stealth.HumanAction, store *storage.Store) *Service {
	return &Service{Stealth: s, Store: store}
}

const MaxNoteLength = 300
const DailyLimit = 20

func (s *Service) Connect(page *rod.Page, profileID int, name string, note string) error {
	logging.Logger.Infof("Starting connection workflow for %s (ID: %d)", name, profileID)

	// 1. Enforce Daily Limit
	count, err := s.Store.GetTodaysRequestCount()
	if err != nil {
		return fmt.Errorf("failed to check daily count: %v", err)
	}
	if count >= DailyLimit {
		logging.Logger.Warnf("Daily limit of %d reached. Skipping %s.", DailyLimit, name)
		return fmt.Errorf("daily limit reached")
	}

	// 2. Check if already requested in DB
	already, _ := s.Store.IsRequested(profileID)
	if already {
		logging.Logger.Infof("Already sent request to %s, skipping", name)
		return nil
	}

	// 3. Skip Navigation - Action directly on Search Results
	logging.Logger.Infof("Locating connection target on current page: %s", name)
	s.Store.LogActivity("Action", fmt.Sprintf("Connecting to %s from Search Results", name))

	// 4. Click Connect Button with precise targeting by data-id
	// Wrap in retry logic to handle potential page load delays or UI state sync issues
	err = retry.WithExponentialBackoff("Click Connect", 3, func() error {
		selector := fmt.Sprintf(".connect-btn[data-id='%d']", profileID)
		btn, err := page.Element(selector)
		if err != nil {
			return fmt.Errorf("connect button not found")
		}

		if !btn.MustVisible() {
			btn.MustScrollIntoView()
		}

		return btn.Click(proto.InputMouseButtonLeft, 1)
	})

	if err != nil {
		logging.Logger.Errorf("Critical: %v", err)
		return err
	}

	s.Stealth.ThinkDelay(800, 1600)

	// 5. Handle Note Modal
	logging.Logger.Info("Applying personalized invitation note...")

	// Enforce character limit
	if len(note) > MaxNoteLength {
		note = note[:MaxNoteLength]
		logging.Logger.Warn("Note truncated to 300 characters")
	}

	if note == "" {
		note = fmt.Sprintf("Hi %s, I saw your profile and would love to connect!", name)
	}

	s.Store.LogActivity("Action", fmt.Sprintf("Typing note for %s", name))
	err = s.Stealth.TypeAction(page, "#note-text", note)
	if err != nil {
		return fmt.Errorf("failed to type note: %v", err)
	}

	s.Stealth.ThinkDelay(1200, 2500)

	// 6. Final Dispatch & Transaction Verification
	logging.Logger.Info("Initializing final handshake dispatch...")
	sendBtn, err := page.Element("#send-note")
	if err != nil {
		return fmt.Errorf("transaction anchor (#send-note) not found: %v", err)
	}

	if !sendBtn.MustVisible() {
		return fmt.Errorf("transaction anchor is obstructed or non-visible")
	}

	s.Store.LogActivity("Action", fmt.Sprintf("Dispatching request to %s", name))
	err = sendBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return fmt.Errorf("failed to finalize handshake: %v", err)
	}

	// Wait for Modal to clear (Verification of success)
	err = page.WaitIdle(time.Second * 5)
	if err != nil {
		logging.Logger.Warn("UI state sync delay detected after dispatch")
	}

	// 7. Telemetry Synchronization & Persistence
	err = s.Store.MarkRequested(profileID, name, "", "")
	if err != nil {
		logging.Logger.Errorf("Critical: Session persistence failure for %s", name)
	}
	s.Store.LogActivity("Success", fmt.Sprintf("Handshake synchronized with %s", name))

	logging.Logger.Infof("Transaction finalized for %s. Pipeline: %d/%d", name, count+1, DailyLimit)
	return nil
}
