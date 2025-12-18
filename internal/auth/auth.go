package auth

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/user/autolink/internal/logging"
	"github.com/user/autolink/internal/stealth"
)

type Manager struct {
	Stealth *stealth.HumanAction
}

func New(s *stealth.HumanAction) *Manager {
	return &Manager{Stealth: s}
}

func (m *Manager) Login(page *rod.Page, username, password, otp string) error {
	logging.Logger.Infof("Attempting login for user: %s", username)

	// Go to home page first to check if session is still valid
	err := page.Navigate("http://localhost:8080/")
	if err != nil {
		return fmt.Errorf("initial navigation failed: %v", err)
	}
	page.WaitLoad()

	// Check if already logged in
	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info: %v", err)
	}
	currURL := info.URL

	if !strings.Contains(currURL, "/login") && !strings.Contains(currURL, "/2fa") {
		logging.Logger.Info("Session still valid, skipping login.")
		return nil
	}

	// Not logged in, go to login page explicitly if needed
	if !strings.Contains(currURL, "/login") {
		err = page.Navigate("http://localhost:8080/login")
		if err != nil {
			return err
		}
		page.WaitLoad()
	}

	// IDENTIFY SECURITY CHECKPOINTS (CAPTCHA)
	if page.MustHas("#captcha-box") || page.MustHas(".captcha") {
		logging.Logger.Warn("SECURITY CHECKPOINT: Captcha detected. Bot requires manual intervention or captcha-solver integration.")
		return fmt.Errorf("captcha detected - automation paused")
	}

	// Fill username
	logging.Logger.Debug("Typing username...")
	el, err := page.Element("#username-field")
	if err != nil {
		return fmt.Errorf("username field not found: %v", err)
	}
	el.MustInput(username)

	// Fill password
	logging.Logger.Debug("Typing password...")
	el, err = page.Element("#password-field")
	if err != nil {
		return fmt.Errorf("password field not found: %v", err)
	}
	el.MustInput(password)

	m.Stealth.ThinkDelay(500, 1500)
	logging.Logger.Debug("Clicking login button...")

	btn, err := page.Element("#login-submit")
	if err != nil {
		return fmt.Errorf("login button not found: %v", err)
	}
	btn.MustClick()

	// Wait for navigation or URL change
	logging.Logger.Info("Waiting for page load...")
	page.WaitLoad()

	// Check for login failures (DETECTION)
	hasError := false
	errEl, err := page.Timeout(1 * time.Second).Element(".error")
	if err == nil && errEl != nil {
		hasError = true
	}

	if hasError {
		errMsg := errEl.MustText()
		return fmt.Errorf("login failed: %s", errMsg)
	}

	postLoginInfo, err := page.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info after login: %v", err)
	}
	currURL = postLoginInfo.URL
	logging.Logger.Infof("Current URL after login attempt: %s", currURL)

	// Identify SECURITY CHECKPOINTS (2FA)
	if strings.Contains(currURL, "/2fa") {
		logging.Logger.Info("Security checkpoint (2FA) detected, solving autonomously...")

		puzzleEl, err := page.Element("#puzzle-text")
		if err != nil {
			return fmt.Errorf("2FA puzzle text not found")
		}
		puzzleText := puzzleEl.MustText()
		logging.Logger.Infof("Puzzle: %s", puzzleText)

		solution, err := solvePuzzle(puzzleText)
		if err != nil {
			return fmt.Errorf("failed to solve security puzzle: %v", err)
		}
		logging.Logger.Infof("Solution calculated: %d", solution)

		logging.Logger.Info("Typing 2FA solution...")
		err = m.Stealth.TypeAction(page, "#otp-field", strconv.Itoa(solution))
		if err != nil {
			return fmt.Errorf("failed to type OTP: %v", err)
		}
		logging.Logger.Info("Typing 2FA solution complete.")

		m.Stealth.ThinkDelay(500, 1000)

		logging.Logger.Info("Looking for 2FA submit button...")
		otpBtn, err := page.Element("#otp-submit")
		if err != nil {
			return fmt.Errorf("2FA submit button not found: %v", err)
		}

		logging.Logger.Info("Clicking 2FA submit button...")
		wait := page.MustWaitNavigation()

		err = otpBtn.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			return fmt.Errorf("failed to click 2FA submit button: %v", err)
		}

		logging.Logger.Info("Waiting for navigation completion...")
		wait()

		logging.Logger.Info("Navigation complete after 2FA.")
		time.Sleep(1 * time.Second)

		// Check for failure on 2FA
		logging.Logger.Info("Checking for 2FA error messages...")
		hasErr, errEl, _ := page.Has(".error")
		if hasErr && errEl != nil {
			errMsg := errEl.MustText()
			return fmt.Errorf("2FA verification failed: %s", errMsg)
		}
	}

	finalInfo, err := page.Info()
	if err != nil {
		return fmt.Errorf("failed to get final page info: %v", err)
	}
	finalURL := finalInfo.URL
	logging.Logger.Infof("Final URL: %s", finalURL)

	// Verify we are logged in (redirected to / or contains no login/2fa)
	if strings.Contains(finalURL, "/login") || strings.Contains(finalURL, "/2fa") {
		return fmt.Errorf("failed to reach dashboard, current URL: %s", finalURL)
	}

	logging.Logger.Info("Authentication successful")
	return nil
}

func solvePuzzle(text string) (int, error) {
	parts := strings.Fields(text)
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid puzzle format: %s", text)
	}

	n1, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	op := parts[1]

	n2, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}

	switch op {
	case "+":
		return n1 + n2, nil
	case "-":
		return n1 - n2, nil
	case "*":
		return n1 * n2, nil
	default:
		return 0, fmt.Errorf("unknown operator: %s", op)
	}
}
