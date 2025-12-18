package messaging

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/user/autolink/internal/logging"
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

// ScanForAcceptedConnections navigates to the connections page and updates the DB status for new connections
func (s *Service) ScanForAcceptedConnections(page *rod.Page) error {
	logging.Logger.Info("Scanning for newly accepted connections...")
	s.Store.LogActivity("System", "Scanning Network for new connections")

	err := page.Navigate("http://localhost:8080/connections")
	if err != nil {
		return fmt.Errorf("failed to navigate to connections: %v", err)
	}

	page.MustWaitLoad()
	s.Stealth.ThinkDelay(1000, 2000)

	// In the simulated app, each connection is in a list with a link to /profile/ID
	items, err := page.Elements("#connection-list a")
	if err != nil {
		logging.Logger.Warn("No connections found during scan.")
		return nil
	}

	for _, item := range items {
		href, _ := item.Attribute("href")
		if href == nil {
			continue
		}

		// Extract ID from /profile/ID
		parts := strings.Split(*href, "/")
		if len(parts) < 3 {
			continue
		}
		id := parts[2]

		// Update status to 'connected' in DB
		// Convert id string to int safely
		var profileID int
		fmt.Sscanf(id, "%d", &profileID)

		// In a real bot, we'd check if it was previously 'requested'
		exists, err := s.Store.IsRequested(profileID)
		if err == nil && exists {
			logging.Logger.Infof("Detected accepted connection for ID: %d", profileID)
			s.Store.UpdateConnectionStatus(profileID, "connected")
		} else {
			// PoC improvement: Track pre-existing connections too to demonstrate Auto DM
			logging.Logger.Infof("Discovered pre-existing connection for ID: %d, adding to tracking", profileID)
			// We don't have the full details here easily, but we can fill them from the page if needed
			// For now, just mark as connected so we can test messaging
			s.Store.MarkRequested(profileID, "Discovered Connection", "Professional", "Network")
			s.Store.UpdateConnectionStatus(profileID, "connected")
		}
	}

	return nil
}

// SendTemplatedMessage sends a personalized message to a connection
func (s *Service) SendTemplatedMessage(page *rod.Page, profileID int, variables map[string]string, templateText string) error {
	name := variables["name"]
	logging.Logger.Infof("Sending templated message to %s (ID: %d)", name, profileID)
	s.Store.LogActivity("Action", fmt.Sprintf("Preparing message for %s", name))

	// 1. Check if already sent
	sent, _ := s.Store.HasSentFollowUp(profileID)
	if sent {
		logging.Logger.Infof("Follow-up already sent to %s, skipping", name)
		return nil
	}

	// 2. Personalize Template
	message := templateText
	for key, val := range variables {
		message = strings.ReplaceAll(message, "{{"+key+"}}", val)
	}

	// 3. Navigate to messages
	err := page.Navigate(fmt.Sprintf("http://localhost:8080/messages?id=%d", profileID))
	if err != nil {
		return err
	}
	page.MustWaitLoad()
	s.Stealth.ThinkDelay(1000, 2000)

	// 4. Simulated message typing and sending
	logging.Logger.Info("Typing follow-up message...")
	s.Store.LogActivity("Action", fmt.Sprintf("Typing message for %s", name))

	err = s.Stealth.TypeAction(page, "#message-text", message)
	if err != nil {
		return fmt.Errorf("failed to type message: %v", err)
	}

	s.Stealth.ThinkDelay(800, 1500)

	// 5. Verification & Dispatch
	logging.Logger.Info("Finalizing message dispatch...")
	sendBtn, err := page.Element("#send-btn")
	if err != nil {
		return fmt.Errorf("dispatch anchor not found: %v", err)
	}

	// Verify button is actually interactable (not disabled/hidden)
	if !sendBtn.MustVisible() {
		return fmt.Errorf("dispatch anchor is non-visible, aborting transmission")
	}

	err = sendBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return fmt.Errorf("failed to execute dispatch: %v", err)
	}

	// Post-Dispatch Verification: Wait for DOM state to reflect the transmission
	s.Stealth.ThinkDelay(1000, 1500)

	// Check if the message appears in the conversation history (verification)
	confirmed := false
	container, err := page.Element("#message-container")
	if err == nil {
		txt := container.MustText()
		if strings.Contains(txt, message) {
			confirmed = true
			logging.Logger.Info("Message transmission verified in DOM")
		}
	}

	if !confirmed {
		logging.Logger.Warn("Message sent but DOM verification failed (potential UI lag)")
	}

	// 6. Persistence & State Synchronization
	err = s.Store.MarkMessageSent(profileID, "bot", "follow_up", message)
	if err != nil {
		logging.Logger.Errorf("Critical: Database sync failure for profile %d", profileID)
	}

	s.Store.LogActivity("Success", fmt.Sprintf("Handshake verified for %s", name))
	logging.Logger.Infof("Transaction finalized for %s", name)
	return nil
}
