package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
	"github.com/timshannon/badgerhold/v4"
)

// ScheduledAnnouncement is a queued announcement waiting to be sent at SendAt.
// Stored in BadgerDB via badgerhold.
type ScheduledAnnouncement struct {
	ID       string    `badgerhold:"key"`
	Users    []string  // Jellyfin user IDs to deliver to
	Subject  string    // Email subject
	Message  string    // Markdown body
	SendAt   time.Time // When to send
	Sent     bool      // True once dispatched
	SentAt   time.Time // When dispatched (zero if not yet)
	CreatedAt time.Time
}

type scheduledAnnouncementDTO struct {
	ID       string    `json:"id,omitempty"`
	Users    []string  `json:"users"`
	Subject  string    `json:"subject"`
	Message  string    `json:"message"`
	SendAt   time.Time `json:"send_at"`
	Sent     bool      `json:"sent"`
	SentAt   time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type scheduledAnnouncementListDTO struct {
	Scheduled []scheduledAnnouncementDTO `json:"scheduled"`
}

// @Summary List queued scheduled announcements.
// @Produce json
// @Success 200 {object} scheduledAnnouncementListDTO
// @Router /users/announce/scheduled [get]
// @Security Bearer
// @tags Users
func (app *appContext) ListScheduledAnnouncements(gc *gin.Context) {
	var stored []ScheduledAnnouncement
	if err := app.storage.db.Find(&stored, (&badgerhold.Query{}).SortBy("SendAt")); err != nil {
		respondAPIError(500, "errorStorage", "STORAGE_QUERY", err.Error(), gc)
		return
	}
	out := make([]scheduledAnnouncementDTO, 0, len(stored))
	for _, s := range stored {
		out = append(out, scheduledAnnouncementDTO{
			ID:        s.ID,
			Users:     s.Users,
			Subject:   s.Subject,
			Message:   s.Message,
			SendAt:    s.SendAt,
			Sent:      s.Sent,
			SentAt:    s.SentAt,
			CreatedAt: s.CreatedAt,
		})
	}
	gc.JSON(200, scheduledAnnouncementListDTO{Scheduled: out})
}

// @Summary Schedule an announcement to be sent at a future time.
// @Produce json
// @Param scheduledAnnouncementDTO body scheduledAnnouncementDTO true "Scheduled announcement"
// @Success 200 {object} scheduledAnnouncementDTO
// @Router /users/announce/scheduled [post]
// @Security Bearer
// @tags Users
func (app *appContext) CreateScheduledAnnouncement(gc *gin.Context) {
	var req scheduledAnnouncementDTO
	if err := gc.BindJSON(&req); err != nil {
		respondAPIError(400, "errorBadRequest", "BAD_REQUEST", err.Error(), gc)
		return
	}
	if req.SendAt.IsZero() || req.SendAt.Before(time.Now()) {
		respondAPIError(400, "errorScheduleInPast", "SCHEDULE_PAST", "send_at must be in the future", gc)
		return
	}
	if req.Subject == "" || req.Message == "" || len(req.Users) == 0 {
		respondAPIError(400, "errorScheduleIncomplete", "SCHEDULE_INCOMPLETE", "users, subject, and message are required", gc)
		return
	}
	rec := ScheduledAnnouncement{
		ID:        shortuuid.New(),
		Users:     req.Users,
		Subject:   req.Subject,
		Message:   req.Message,
		SendAt:    req.SendAt,
		CreatedAt: time.Now(),
	}
	if err := app.storage.db.Insert(rec.ID, rec); err != nil {
		respondAPIError(500, "errorStorage", "STORAGE_INSERT", err.Error(), gc)
		return
	}
	req.ID = rec.ID
	req.CreatedAt = rec.CreatedAt
	gc.JSON(200, req)
}

// @Summary Cancel a queued scheduled announcement.
// @Param id path string true "Scheduled announcement ID"
// @Success 200 {object} boolResponse
// @Failure 404 {object} stringResponse
// @Router /users/announce/scheduled/{id} [delete]
// @Security Bearer
// @tags Users
func (app *appContext) DeleteScheduledAnnouncement(gc *gin.Context) {
	id := gc.Param("id")
	var rec ScheduledAnnouncement
	if err := app.storage.db.Get(id, &rec); err != nil {
		gc.String(http.StatusNotFound, "not found")
		return
	}
	if err := app.storage.db.Delete(id, ScheduledAnnouncement{}); err != nil {
		respondAPIError(500, "errorStorage", "STORAGE_DELETE", err.Error(), gc)
		return
	}
	respondBool(200, true, gc)
}

// scheduledAnnounceJob is run on every daemon tick: dispatches any announcements
// whose SendAt is now or in the past, then deletes them.
func scheduledAnnounceJob(app *appContext) {
	if !messagesEnabled {
		return
	}
	var due []ScheduledAnnouncement
	if err := app.storage.db.Find(&due, badgerhold.Where("SendAt").Le(time.Now()).And("Sent").Eq(false)); err != nil {
		app.debug.Printf("scheduled announce daemon: query failed: %v", err)
		return
	}
	if len(due) == 0 {
		return
	}
	for _, s := range due {
		app.debug.Printf("scheduled announce daemon: dispatching %s (%d users)", s.ID, len(s.Users))
		// Use the same path as the live Announce handler.
		msg, err := app.email.construct(AnnouncementCustomContent(s.Subject), CustomContent{
			Enabled: true,
			Content: s.Message,
		}, map[string]any{"username": ""})
		if err != nil {
			app.err.Printf("scheduled announce: construct failed for %s: %v", s.ID, err)
			continue
		}
		if err := app.sendByID(msg, s.Users...); err != nil {
			app.err.Printf("scheduled announce: send failed for %s: %v", s.ID, err)
			continue
		}
		s.Sent = true
		s.SentAt = time.Now()
		_ = app.storage.db.Update(s.ID, s)
	}
}

// newScheduledAnnounceDaemon returns a GenericDaemon that polls for due
// scheduled announcements once per minute.
func newScheduledAnnounceDaemon(app *appContext) *GenericDaemon {
	d := NewGenericDaemon(60*time.Second, app, scheduledAnnounceJob)
	d.Name("Scheduled Announcements")
	return d
}
