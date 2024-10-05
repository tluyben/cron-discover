package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Cron struct {
	ID              int64
	WorkspaceID     int64
	Name            string
	Description     string
	CronExpression  string
}

type Webhook struct {
	ID     int64
	CronID int64
	URL    string
}

func createCron(db *sql.DB, c *Cron) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec("INSERT INTO crons (workspace_id, name, description, cron_expression) VALUES (?, ?, ?, ?)",
		c.WorkspaceID, c.Name, c.Description, c.CronExpression)
	if err != nil {
		return err
	}

	c.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}

	return tx.Commit()
}

func listCrons(db *sql.DB, workspaceID int64) ([]Cron, error) {
	rows, err := db.Query("SELECT id, workspace_id, name, description, cron_expression FROM crons WHERE workspace_id = ?", workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var crons []Cron
	for rows.Next() {
		var c Cron
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.Description, &c.CronExpression); err != nil {
			return nil, err
		}
		crons = append(crons, c)
	}

	return crons, rows.Err()
}

func deleteCron(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM crons WHERE id = ?", id)
	return err
}

func addWebhook(db *sql.DB, cronID int64, url string) error {
	_, err := db.Exec("INSERT INTO webhooks (cron_id, url) VALUES (?, ?)", cronID, url)
	return err
}

func listWebhooks(db *sql.DB, cronID int64) ([]Webhook, error) {
	rows, err := db.Query("SELECT id, cron_id, url FROM webhooks WHERE cron_id = ?", cronID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.CronID, &w.URL); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}

	return webhooks, rows.Err()
}

func removeWebhook(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM webhooks WHERE id = ?", id)
	return err
}

func notifyWebhooks(db *sql.DB, cronID int64) error {
	webhooks, err := listWebhooks(db, cronID)
	if err != nil {
		return err
	}

	for _, webhook := range webhooks {
		go func(url string) {
			resp, err := http.Post(url, "application/json", nil)
			if err != nil {
				log.Printf("Error notifying webhook %s: %v", url, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				log.Printf("Webhook %s returned non-2xx status code: %d", url, resp.StatusCode)
			}
		}(webhook.URL)
	}

	return nil
}

func getWorkspaceDB(workspacesDir string, workspaceID int64) (*sql.DB, error) {
	dbPath := filepath.Join(workspacesDir, fmt.Sprintf("%d", workspaceID), "cron.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	
	// Initialize the workspace database if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cron_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cron_id INTEGER NOT NULL,
			execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}
	
	return db, nil
}