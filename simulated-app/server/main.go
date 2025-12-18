package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/user/autolink/internal/storage"
)

var db *sql.DB
var store *storage.Store

var currentAnswer int

type Profile struct {
	ID         int
	Name       string
	Title      string
	Company    string
	Location   string
	Connected  bool
	About      string
	Experience []Experience
	Education  []Education
	Skills     []string
}

type Experience struct {
	Title       string
	Company     string
	Duration    string
	Description string
}

type Education struct {
	School string
	Degree string
	Year   string
}

var profiles = []Profile{
	{
		ID:        1,
		Name:      "Alice Smith",
		Title:     "Staff Software Engineer",
		Company:   "TechNova",
		Location:  "San Francisco, CA",
		Connected: false,
		About:     "Passionate about building scalable systems and mentoring engineers. 10+ years of experience in distributed systems and cloud architecture.",
		Experience: []Experience{
			{"Staff Software Engineer", "TechNova", "2020 - Present", "Leading the core infrastructure team, optimizing high-throughput data pipelines."},
			{"Senior Software Engineer", "CloudScale", "2016 - 2020", "Architected and deployed microservices using Go and Kubernetes."},
		},
		Education: []Education{
			{"Stanford University", "MS in Computer Science", "2016"},
		},
		Skills: []string{"Go", "Distributed Systems", "Kubernetes", "Cloud Architecture", "System Design"},
	},
	{
		ID:        2,
		Name:      "Bob Jones",
		Title:     "Product Executive",
		Company:   "FinLeap",
		Location:  "New York, NY",
		Connected: false,
		About:     "Strategic product leader focused on fintech innovation. Expert in scaling product teams from 0 to 1.",
		Experience: []Experience{
			{"Product Executive", "FinLeap", "2021 - Present", "Overseeing product strategy for the investment banking division."},
			{"Director of Product", "PayFast", "2017 - 2021", "Launched the mobile payment platform used by millions."},
		},
		Education: []Education{
			{"Harvard Business School", "MBA", "2017"},
		},
		Skills: []string{"Product Strategy", "Fintech", "Leadership", "Agile Roadmap", "Business Development"},
	},
	{
		ID:        3,
		Name:      "Charlie Brown",
		Title:     "Head of Data Science",
		Company:   "OrbitAI",
		Location:  "Seattle, WA",
		Connected: false,
		About:     "Applying machine learning to solve complex problems in aerospace. Research enthusiast and open-source contributor.",
		Experience: []Experience{
			{"Head of Data Science", "OrbitAI", "2019 - Present", "Building the next generation of predictive maintenance for satellites."},
			{"Senior ML Engineer", "DataFlow", "2015 - 2019", "Developed NLP models for consumer sentiment analysis."},
		},
		Education: []Education{
			{"MIT", "PhD in Artificial Intelligence", "2015"},
		},
		Skills: []string{"Machine Learning", "Python", "PyTorch", "NLP", "Big Data"},
	},
	{
		ID:        4,
		Name:      "Diana Prince",
		Title:     "Security Architect",
		Company:   "ShieldCorp",
		Location:  "Washington, DC",
		Connected: false,
		About:     "Cybersecurity expert with a focus on zero-trust architectures. Dedicated to protecting critical infrastructure.",
		Experience: []Experience{
			{"Security Architect", "ShieldCorp", "2018 - Present", "Designing multi-layered security protocols for government agencies."},
			{"Security Analyst", "GlobalDefense", "2014 - 2018", "Monitored and mitigated large-scale DDoS attacks."},
		},
		Education: []Education{
			{"Georgetown University", "BS in Cybersecurity", "2014"},
		},
		Skills: []string{"Cybersecurity", "Zero Trust", "Network Security", "Risk Assessment", "Ethical Hacking"},
	},
	{
		ID:        5,
		Name:      "Ethan Hunt",
		Title:     "Infrastructure Lead",
		Company:   "DeepCloud",
		Location:  "Austin, TX",
		Connected: true,
		About:     "Specializing in mission-critical infrastructure and high-availability systems. I solve the impossible.",
		Experience: []Experience{
			{"Infrastructure Lead", "DeepCloud", "2017 - Present", "Guaranteeing 99.999% uptime for global cloud services."},
			{"DevOps Engineer", "IronSafe", "2012 - 2017", "Automated deployment pipelines for financial applications."},
		},
		Education: []Education{
			{"University of Texas", "BS in Computer Engineering", "2012"},
		},
		Skills: []string{"DevOps", "Infrastructure as Code", "Terraform", "AWS", "Site Reliability"},
	},
	{
		ID:        6,
		Name:      "Fiona Glenanne",
		Title:     "Fullstack Developer",
		Company:   "WebFlow",
		Location:  "Miami, FL",
		Connected: false,
		About:     "Explosive growth specialist. I build fast, secure, and beautiful web applications.",
		Experience: []Experience{
			{"Fullstack Developer", "WebFlow", "2019 - Present", "Developing high-performance UI components and backend services."},
		},
		Education: []Education{
			{"University of Miami", "BS in IT", "2018"},
		},
		Skills: []string{"React", "Node.js", "SQL", "Tailwind", "Firebase"},
	},
	{
		ID:        7,
		Name:      "George Costanza",
		Title:     "Latex Salesman",
		Company:   "Vandelay Industries",
		Location:  "New York, NY",
		Connected: false,
		About:     "Experienced professional in import/export and latex sales. Architect enthusiast.",
		Experience: []Experience{
			{"Latex Salesman", "Vandelay Industries", "1994 - Present", "Managing the entire latex division (which is just me)."},
		},
		Education: []Education{
			{"Queens College", "BA", "1980"},
		},
		Skills: []string{"Sales", "Import/Export", "Latex", "Lying", "Napping"},
	},
}

func main() {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./simulated-app/static"))))

	mux.HandleFunc("/", logRequest(handleFeed))
	mux.HandleFunc("/login", logRequest(handleLogin))
	mux.HandleFunc("/logout", logRequest(handleLogout))
	mux.HandleFunc("/2fa", logRequest(handle2FA))
	mux.HandleFunc("/search", logRequest(handleSearch))
	mux.HandleFunc("/profile/", logRequest(handleProfile))
	mux.HandleFunc("/connections", logRequest(handleConnections))
	mux.HandleFunc("/messages", logRequest(handleMessages))
	mux.HandleFunc("/api/connect", logRequest(handleConnectAPI))
	mux.HandleFunc("/api/logs", logRequest(handleLogs))
	mux.HandleFunc("/api/messages", logRequest(handleSentMessages))
	mux.HandleFunc("/api/send-message", logRequest(handleSendMessageAPI))
	mux.HandleFunc("/api/get-messages", logRequest(handleGetMessagesAPI))

	var err error
	db, err = sql.Open("sqlite3", "./autolink.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	store, _ = storage.New("./autolink.db")

	fmt.Println("Simulated LinkedIn-like app running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func logRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		h(w, r)
	}
}

func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) {
	t, err := template.ParseFiles(
		"./simulated-app/templates/layout.html",
		"./simulated-app/templates/"+tmpl+".html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	authenticated := IsAuthenticated(r)

	sentRequests := "--"
	if authenticated && db != nil {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM connections").Scan(&count)
		if err == nil {
			sentRequests = strconv.Itoa(count)
		}
	}

	// Create a wrapper data map to include base layout info
	todaysProgress := 0
	if authenticated && db != nil {
		db.QueryRow("SELECT COUNT(*) FROM connections WHERE created_at >= date('now', 'start of day')").Scan(&todaysProgress)
	}

	wrappedData := map[string]interface{}{
		"Authenticated":  authenticated,
		"Data":           data,
		"SentRequests":   sentRequests,
		"TodaysProgress": todaysProgress,
	}

	t.ExecuteTemplate(w, "layout", wrappedData)
}

func handleFeed(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	renderTemplate(w, r, "feed", nil)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if IsAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "admin" && password == "password123" {
			http.Redirect(w, r, "/2fa", http.StatusFound)
			return
		}
		renderTemplate(w, r, "login", map[string]string{"Error": "Invalid credentials"})
		return
	}
	renderTemplate(w, r, "login", nil)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func handle2FA(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		code := r.FormValue("code")
		if code == strconv.Itoa(currentAnswer) {
			http.SetCookie(w, &http.Cookie{
				Name:  "session_id",
				Value: "mock_session_token",
				Path:  "/",
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		// On error, let the GET handler generate a new puzzle
		http.Redirect(w, r, "/2fa?error=true", http.StatusFound)
		return
	}

	errorMsg := ""
	if r.URL.Query().Get("error") == "true" {
		errorMsg = "Incorrect answer, try again."
	}

	num1 := rand.Intn(20) + 1
	num2 := rand.Intn(20) + 1
	ops := []string{"+", "-", "*"}
	op := ops[rand.Intn(len(ops))]

	switch op {
	case "+":
		currentAnswer = num1 + num2
	case "-":
		currentAnswer = num1 - num2
	case "*":
		currentAnswer = num1 * num2
	}
	puzzle := fmt.Sprintf("%d %s %d", num1, op, num2)

	renderTemplate(w, r, "2fa", map[string]interface{}{"Puzzle": puzzle, "Error": errorMsg})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	query := r.URL.Query().Get("q")
	results := []Profile{}
	for _, p := range profiles {
		if query == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(query)) || strings.Contains(strings.ToLower(p.Title), strings.ToLower(query)) {
			results = append(results, p)
		}
	}
	renderTemplate(w, r, "search", map[string]interface{}{
		"Query":   query,
		"Results": results,
	})
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/profile/")
	id, _ := strconv.Atoi(idStr)
	var found *Profile
	for _, p := range profiles {
		if p.ID == id {
			found = &p
			break
		}
	}
	renderTemplate(w, r, "profile", found)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	connected := []Profile{}
	for _, p := range profiles {
		if p.Connected {
			connected = append(connected, p)
		}
	}
	renderTemplate(w, r, "connections", connected)
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	connected := []Profile{}
	for _, p := range profiles {
		if p.Connected {
			connected = append(connected, p)
		}
	}
	renderTemplate(w, r, "messages", map[string]interface{}{
		"Connections": connected,
		"CurrentID":   r.URL.Query().Get("id"),
	})
}

func handleConnectAPI(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := r.FormValue("id")
	id, _ := strconv.Atoi(idStr)
	note := r.FormValue("note")
	for i, p := range profiles {
		if p.ID == id {
			profiles[i].Connected = true

			// Persist in DB
			if store != nil {
				store.MarkRequested(id, p.Name, p.Title, p.Company)
				store.UpdateConnectionStatus(id, "connected")

				logMsg := fmt.Sprintf("🤝 Protocol: Handshake with %s established", p.Name)
				if note != "" {
					logMsg += " [+INVITATION]"
				}
				store.LogActivity("SYNC", logMsg)
			}

			fmt.Fprintf(w, "Connected to %s", p.Name)

			// Auto-greeting logic
			go func(profileID int, name, company string) {
				time.Sleep(1500 * time.Millisecond) // Simulate bit of delay
				if store != nil {
					content := fmt.Sprintf("Hi %s, thanks for connecting! I'm interested in your work at %s. Hope you're having a great day!", name, company)
					store.MarkMessageSent(profileID, "bot", "auto_greeting", content)
					store.LogActivity("Auto-Greeting", fmt.Sprintf("Sent to %s", name))
				}
			}(id, p.Name, p.Company)

			return
		}
	}
	http.Error(w, "Profile not found", http.StatusNotFound)
}

func handleSendMessageAPI(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	profileID, _ := strconv.Atoi(r.FormValue("profile_id"))
	content := r.FormValue("content")

	if content == "" || profileID == 0 {
		http.Error(w, "Missing content or profile_id", http.StatusBadRequest)
		return
	}

	if store != nil {
		err := store.MarkMessageSent(profileID, "user", "dm", content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Find profile name for logging
		var name string
		for _, p := range profiles {
			if p.ID == profileID {
				name = p.Name
				break
			}
		}
		store.LogActivity("Message sent", fmt.Sprintf("DM to %s: %s", name, content))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleGetMessagesAPI(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	profileID, _ := strconv.Atoi(r.URL.Query().Get("profile_id"))
	if profileID == 0 {
		http.Error(w, "Missing profile_id", http.StatusBadRequest)
		return
	}

	msgs, err := store.GetMessagesForProfile(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func IsAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session_id")
	return err == nil && cookie.Value == "mock_session_token"
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "DB not initialized", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT action_type, metadata, created_at FROM activity_log ORDER BY id DESC LIMIT 15")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type LogEntry struct {
		Action   string `json:"action"`
		Metadata string `json:"metadata"`
		Time     string `json:"time"`
	}
	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		var t time.Time
		rows.Scan(&l.Action, &l.Metadata, &t)
		l.Time = t.Format("15:04:05")
		logs = append(logs, l)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
func handleSentMessages(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "DB not initialized", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT profile_id, message_type, content, sent_at FROM messages ORDER BY id DESC LIMIT 10")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type MsgEntry struct {
		ProfileID int    `json:"profile_id"`
		Type      string `json:"type"`
		Content   string `json:"content"`
		Time      string `json:"time"`
	}
	var msgs []MsgEntry
	for rows.Next() {
		var m MsgEntry
		var t time.Time
		rows.Scan(&m.ProfileID, &m.Type, &m.Content, &t)
		m.Time = t.Format("15:04")
		msgs = append(msgs, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}
