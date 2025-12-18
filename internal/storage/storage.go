package storage

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS connections (
			profile_id INTEGER PRIMARY KEY,
			name TEXT,
			title TEXT,
			company TEXT,
			status TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS activity_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action_type TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id INTEGER,
			sender TEXT, -- 'bot' or 'user'
			message_type TEXT, -- 'follow_up', 'outreach', etc.
			content TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) MarkRequested(profileID int, name, title, company string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO connections (profile_id, name, title, company, status) VALUES (?, ?, ?, ?, ?)",
		profileID, name, title, company, "requested",
	)
	return err
}

func (s *Store) IsRequested(profileID int) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM connections WHERE profile_id = ?", profileID).Scan(&count)
	return count > 0, err
}

func (s *Store) LogActivity(actionType, metadata string) error {
	_, err := s.db.Exec("INSERT INTO activity_log (action_type, metadata) VALUES (?, ?)", actionType, metadata)
	return err
}

func (s *Store) GetTodaysRequestCount() (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM connections 
		WHERE status = 'requested' 
		AND created_at >= date('now', 'start of day')
	`).Scan(&count)
	return count, err
}

func (s *Store) MarkMessageSent(profileID int, sender, msgType, content string) error {
	_, err := s.db.Exec("INSERT INTO messages (profile_id, sender, message_type, content) VALUES (?, ?, ?, ?)", profileID, sender, msgType, content)
	return err
}

func (s *Store) HasSentFollowUp(profileID int) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM messages WHERE profile_id = ? AND message_type = 'follow_up'", profileID).Scan(&count)
	return count > 0, err
}

func (s *Store) GetPendingFollowUps() ([]Connection, error) {
	rows, err := s.db.Query(`
		SELECT profile_id, name, company FROM connections 
		WHERE status = 'connected' 
		AND profile_id NOT IN (SELECT profile_id FROM messages WHERE message_type = 'follow_up')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Connection
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ProfileID, &c.Name, &c.Company); err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}

func (s *Store) UpdateConnectionStatus(profileID int, status string) error {
	_, err := s.db.Exec("UPDATE connections SET status = ? WHERE profile_id = ?", status, profileID)
	return err
}

func (s *Store) GetMessagesForProfile(profileID int) ([]Message, error) {
	rows, err := s.db.Query("SELECT sender, content, sent_at FROM messages WHERE profile_id = ? ORDER BY sent_at ASC", profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var t time.Time
		if err := rows.Scan(&m.Sender, &m.Content, &t); err != nil {
			return nil, err
		}
		m.Time = t.Format("15:04")
		msgs = append(msgs, m)
	}
	return msgs, nil
}

type Message struct {
	Sender  string
	Content string
	Time    string
}

type Connection struct {
	ProfileID int
	Name      string
	Company   string
	Status    string
}
