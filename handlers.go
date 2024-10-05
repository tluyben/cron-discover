package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
)

type CronHandler struct {
	db            *sql.DB
	workspacesDir string
	scheduler     *cron.Cron
}

func (h *CronHandler) CreateCron(w http.ResponseWriter, r *http.Request) {
	var c Cron
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := createCron(h.db, &c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.scheduleCron(c)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func (h *CronHandler) ListCrons(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := strconv.ParseInt(r.URL.Query().Get("workspace_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid workspace_id", http.StatusBadRequest)
		return
	}

	crons, err := listCrons(h.db, workspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(crons)
}

func (h *CronHandler) DeleteCron(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid cron ID", http.StatusBadRequest)
		return
	}

	if err := deleteCron(h.db, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CronHandler) AddWebhook(w http.ResponseWriter, r *http.Request) {
	cronID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid cron ID", http.StatusBadRequest)
		return
	}

	var webhook Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := addWebhook(h.db, cronID, webhook.URL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CronHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	cronID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid cron ID", http.StatusBadRequest)
		return
	}

	webhooks, err := listWebhooks(h.db, cronID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(webhooks)
}

func (h *CronHandler) RemoveWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID, err := strconv.ParseInt(mux.Vars(r)["webhookId"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	if err := removeWebhook(h.db, webhookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CronHandler) StartCronScheduler(ctx context.Context) {
	h.scheduler = cron.New()
	h.scheduler.Start()

	go func() {
		<-ctx.Done()
		h.scheduler.Stop()
	}()

	// Load existing crons from the database and schedule them
	rows, err := h.db.Query("SELECT id, workspace_id, name, description, cron_expression FROM crons")
	if err != nil {
		log.Printf("Error loading crons: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var c Cron
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.Description, &c.CronExpression); err != nil {
			log.Printf("Error scanning cron row: %v", err)
			continue
		}
		h.scheduleCron(c)
	}
}

func (h *CronHandler) scheduleCron(c Cron) {
	_, err := h.scheduler.AddFunc(c.CronExpression, func() {
		workspaceDB, err := getWorkspaceDB(h.workspacesDir, c.WorkspaceID)
		if err != nil {
			log.Printf("Error opening workspace database for cron %d: %v", c.ID, err)
			return
		}
		defer workspaceDB.Close()

		// Log the execution
		_, err = workspaceDB.Exec("INSERT INTO cron_logs (cron_id, status) VALUES (?, ?)", c.ID, "started")
		if err != nil {
			log.Printf("Error logging cron execution start for cron %d: %v", c.ID, err)
		}

		// Notify webhooks
		if err := notifyWebhooks(h.db, c.ID); err != nil {
			log.Printf("Error notifying webhooks for cron %d: %v", c.ID, err)
			workspaceDB.Exec("UPDATE cron_logs SET status = ? WHERE cron_id = ? AND status = ? ORDER BY execution_time DESC LIMIT 1", 
				"failed", c.ID, "started")
		} else {
			workspaceDB.Exec("UPDATE cron_logs SET status = ? WHERE cron_id = ? AND status = ? ORDER BY execution_time DESC LIMIT 1", 
				"completed", c.ID, "started")
		}
	})
	if err != nil {
		log.Printf("Error scheduling cron %d: %v", c.ID, err)
	}
}