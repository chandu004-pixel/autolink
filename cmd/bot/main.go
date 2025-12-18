package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/user/autolink/internal/auth"
	"github.com/user/autolink/internal/browser"
	"github.com/user/autolink/internal/config"
	"github.com/user/autolink/internal/connect"
	"github.com/user/autolink/internal/logging"
	"github.com/user/autolink/internal/messaging"
	"github.com/user/autolink/internal/search"
	"github.com/user/autolink/internal/stealth"
	"github.com/user/autolink/internal/storage"
)

func main() {
	// 1. Initialize Logging & Config
	logging.Init()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Initialize Storage
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		logging.Logger.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	// 3. Initialize Browser
	client, err := browser.New(cfg.Headless)
	if err != nil {
		logging.Logger.Fatalf("failed to init browser: %v", err)
	}
	defer client.Close()

	// 4. Initialize Services
	human := stealth.New(client.Browser)
	authMgr := auth.New(human)
	searchSvc := search.New(human)
	connectSvc := connect.New(human, store)
	messagingSvc := messaging.New(human, store)

	// 5. Setup Signal Handling for Clean Exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 6. Run Automation Flow
	go func() {
		logging.Logger.Info("Initializing Autolink Intelligence Core...")
		logging.Logger.Info("Synchronizing with Local DB: autolink.db")
		logging.Logger.Info("Applying Stealth Profiling (Fingerprint: x86_64, Chrome/121)")

		page, err := client.NewStealthPage(cfg.AppURL)
		if err != nil {
			logging.Logger.Errorf("FATAL: Bridge engagement failed: %v", err)
			return
		}
		logging.Logger.Info("Secure Bridge Established to LinkSim Environment")

		// Login
		err = authMgr.Login(page, cfg.Username, cfg.Password, cfg.OTP)
		if err != nil {
			logging.Logger.Errorf("Login failed: %v", err)
			return
		}

		// Search for all profiles to demonstrate skipping logic
		results, err := searchSvc.ExecuteSearch(page, "")
		if err != nil {
			logging.Logger.Errorf("Search failed: %v", err)
			return
		}

		// Process Results
		for i, profile := range results {
			// Personalized Greeting Template
			connectTemplate := "Hi {{name}}, I saw your profile and would love to connect! I'm interested in your work at {{company}}."
			vars := map[string]string{
				"name":    profile.Name,
				"company": profile.Company,
			}
			if vars["company"] == "" {
				vars["company"] = profile.Title
			}

			if profile.Connected {
				logging.Logger.Infof("Profile %s is already connected. Sending greeting DM...", profile.Name)
				store.LogActivity("Action", fmt.Sprintf("Greeting %s (already connected)", profile.Name))

				err = messagingSvc.SendTemplatedMessage(page, profile.ID, vars, "Hi {{name}}, great to see you in my network! How are things at {{company}}?")
				if err != nil {
					logging.Logger.Errorf("Failed to greet %s: %v", profile.Name, err)
				}
			} else {
				// Connect Workflow
				// Personalize the connection note
				note := connectTemplate
				note = strings.ReplaceAll(note, "{{name}}", profile.Name)
				note = strings.ReplaceAll(note, "{{company}}", vars["company"])

				err = connectSvc.Connect(page, profile.ID, profile.Name, note)
				if err != nil {
					logging.Logger.Errorf("Failed to connect with %s: %v", profile.Name, err)
				} else {
					// In the simulation, they are connected immediately.
					// Greet them in DM as requested.
					logging.Logger.Infof("Connection requested for %s. Sending follow-up DM...", profile.Name)
					err = messagingSvc.SendTemplatedMessage(page, profile.ID, vars, "Hi {{name}}, thanks for the connection! Excited to follow your work.")
					if err != nil {
						logging.Logger.Errorf("Failed to send follow-up DM to %s: %v", profile.Name, err)
					}
				}
			}

			// If we are not at the end, and we navigated away (to messages), go back to search
			if i < len(results)-1 {
				logging.Logger.Info("Returning to search results for next target...")
				err = page.Navigate(cfg.AppURL + "/search") // Use AppURL/search to be safe
				if err != nil {
					logging.Logger.Errorf("failed to return to search: %v", err)
					break
				}
				page.MustWaitLoad()
				// We might need to re-execute the search if the page state was lost,
				// but in our simulation, the search results might persist or we can just navigate to /search?q=...
				// For simplicity, let's assume /search shows the list.
			}

			// Artificial Cooldown between actions
			logging.Logger.Infof("Cooling down for %d seconds...", cfg.CooldownSeconds)
			time.Sleep(time.Duration(cfg.CooldownSeconds) * time.Second)
		}

		// 7. Messaging & Follow-ups
		logging.Logger.Info("Checking for accepted connections and sending follow-ups...")

		err = messagingSvc.ScanForAcceptedConnections(page)
		if err != nil {
			logging.Logger.Errorf("Failed to scan for accepted connections: %v", err)
		}

		pending, err := store.GetPendingFollowUps()
		if err == nil {
			logging.Logger.Infof("Found %d pending follow-ups", len(pending))
			for _, conn := range pending {
				template := "Hi {{name}}, great to connect with you! I noticed your work at {{company}} - would love to keep in touch."
				vars := map[string]string{
					"name":    conn.Name,
					"company": conn.Company,
				}

				err = messagingSvc.SendTemplatedMessage(page, conn.ProfileID, vars, template)
				if err != nil {
					logging.Logger.Errorf("Failed to send follow-up to %s: %v", conn.Name, err)
				}
				time.Sleep(time.Duration(cfg.CooldownSeconds) * time.Second)
			}
		}

		logging.Logger.Info("Automation flow completed.")
	}()

	<-sigChan
	logging.Logger.Info("Shutting down...")
}
