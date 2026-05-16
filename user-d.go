package main

import (
	"time"

	lm "github.com/jay739/omnifin/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
)

func newUserDaemon(interval time.Duration, app *appContext) *GenericDaemon {
	preExpiryCutoffDays := app.config.Section("user_expiry").Key("send_reminder_n_days_before").StringsWithShadows("|")
	var as *DayTimerSet
	if len(preExpiryCutoffDays) > 0 {
		as = NewDayTimerSet(preExpiryCutoffDays, -24*time.Hour)
	}
	d := NewGenericDaemon(interval, app,
		func(app *appContext) {
			app.checkUsers(as)
		},
	)
	d.Name("User daemon")
	return d
}

const (
	ExpiryModeDisable = iota
	ExpiryModeDelete
)

func (app *appContext) checkUsers(remindBeforeExpiry *DayTimerSet) {
	if len(app.storage.GetUserExpiries()) == 0 {
		return
	}

	app.info.Println(lm.CheckUserExpiries)
	users, err := app.jf.GetUsers(false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		return
	}
	expiryMode := ExpiryModeDisable
	if app.config.Section("user_expiry").Key("behaviour").MustString("disable_user") == "delete_user" {
		expiryMode = ExpiryModeDelete
	}

	deleteAfterPeriod := app.config.Section("user_expiry").Key("delete_expired_after_days").MustInt(0)
	if expiryMode == ExpiryModeDelete {
		deleteAfterPeriod = 0
	}

	shouldContact := messagesEnabled && app.config.Section("user_expiry").Key("send_email").MustBool(true)

	// Use a map to speed up checking for deleted users later
	// FIXME: Maybe expose MediaBrowser.usersByID in some way and use that instead.
	userExists := map[string]mediabrowser.User{}
	for _, user := range users {
		userExists[user.ID] = user
	}

	shouldInvalidateCache := false

	// Activity-based expiry auto-extension. If enabled, when a user has logged into
	// Jellyfin within `auto_extend_if_active_within_days` and their expiry is within
	// `auto_extend_window_days`, push the expiry out by `auto_extend_by_days`.
	autoExtendEnabled := app.config.Section("user_expiry").Key("auto_extend_on_activity").MustBool(false)
	autoExtendActiveWithinDays := app.config.Section("user_expiry").Key("auto_extend_if_active_within_days").MustInt(7)
	autoExtendWindowDays := app.config.Section("user_expiry").Key("auto_extend_window_days").MustInt(14)
	autoExtendByDays := app.config.Section("user_expiry").Key("auto_extend_by_days").MustInt(30)

	for _, expiry := range app.storage.GetUserExpiries() {
		id := expiry.JellyfinID
		user, ok := userExists[id]
		if !ok {
			app.info.Printf(lm.DeleteExpiryForOldUser, id)
			app.storage.DeleteUserExpiryKey(expiry.JellyfinID)
			continue
		}

		// Auto-extend before checking whether expired: a recently active user shouldn't
		// be marked expired and disabled in the same daemon tick.
		if autoExtendEnabled && !expiry.DeleteAfterPeriod && !user.LastActivityDate.Time.IsZero() {
			activeCutoff := time.Now().Add(-time.Duration(autoExtendActiveWithinDays) * 24 * time.Hour)
			expiryWindow := time.Now().Add(time.Duration(autoExtendWindowDays) * 24 * time.Hour)
			if user.LastActivityDate.Time.After(activeCutoff) && expiry.Expiry.Before(expiryWindow) && expiry.Expiry.After(time.Now()) {
				oldExpiry := expiry.Expiry
				expiry.Expiry = expiry.Expiry.AddDate(0, 0, autoExtendByDays)
				expiry.LastNotified = time.Time{} // reset reminder window so the user gets a fresh notice
				app.storage.SetUserExpiryKey(user.ID, expiry)
				app.info.Printf("auto-extended expiry for %s: %s -> %s (active within %dd)", user.Name, oldExpiry.Format("2006-01-02"), expiry.Expiry.Format("2006-01-02"), autoExtendActiveWithinDays)
				app.fireWebhook("expiry_extended", map[string]any{
					"user_id":    user.ID,
					"username":   user.Name,
					"old_expiry": oldExpiry.Format(time.RFC3339),
					"new_expiry": expiry.Expiry.Format(time.RFC3339),
					"reason":     "activity_auto_extend",
				})
				shouldInvalidateCache = true
			}
		}

		if !time.Now().After(expiry.Expiry) {
			if shouldContact && remindBeforeExpiry != nil {
				app.debug.Printf("Checking for expiry reminder timers")
				duration := remindBeforeExpiry.Check(expiry.Expiry, expiry.LastNotified)
				if duration != 0 {
					expiry.LastNotified = time.Now()
					app.storage.SetUserExpiryKey(user.ID, expiry)
					name := app.getAddressOrName(user.ID)
					// Skip blank contact info
					if name == "" {
						continue
					}
					msg, err := app.email.constructExpiryReminder(user.Name, expiry.Expiry, false)
					if err != nil {
						app.err.Printf(lm.FailedConstructExpiryReminderMessage, user.ID, err)
					} else if err := app.sendByID(msg, user.ID); err != nil {
						app.err.Printf(lm.FailedSendExpiryReminderMessage, user.ID, name, err)
					} else {
						app.info.Printf(lm.SentExpiryReminderMessage, user.ID, name)
					}
				}
			}
			continue
		}

		// True when "Delete after period" enabled and this user's account has already expired.
		alreadyExpired := false
		// True when the user has expired and N days has passed for them to be deleted.
		alreadyExpiredShouldDelete := false
		if expiry.DeleteAfterPeriod {
			// Delete hanging expiries after the admin disables "delete after N days"
			if deleteAfterPeriod <= 0 {
				app.storage.DeleteUserExpiryKey(user.ID)
				continue
			}
			alreadyExpired = true
			alreadyExpiredShouldDelete = time.Now().After(expiry.Expiry.AddDate(0, 0, deleteAfterPeriod))
			if !alreadyExpiredShouldDelete {
				continue
			}
		}

		// Record activity
		activity := Activity{
			UserID:     id,
			SourceType: ActivityDaemon,
			Time:       time.Now(),
			Type:       ActivityUnknown,
		}

		// To save you the brain power: these conditions are fine because of all the "continue"s above us, no further checks are needed.
		if expiryMode == ExpiryModeDelete || alreadyExpiredShouldDelete {
			app.info.Printf(lm.DeleteExpiredUser, user.Name)
			deleted := false
			err, deleted = app.DeleteUser(user)
			// Silence unimportant errors
			if deleted {
				err = nil
			}
			activity.Type = ActivityDeletion
			// Store the user name, since there's no longer a user ID to reference back to
			activity.Value = user.Name
			app.InvalidateUserCaches()
		} else {
			app.info.Printf(lm.DisableExpiredUser, user.Name)
			// Admins can't be disabled
			// so they're not an admin anymore, sorry
			user.Policy.IsAdministrator = false
			err, _, _ = app.SetUserDisabled(user, true)
			activity.Type = ActivityDisabled
			app.InvalidateUserCaches()
		}
		if err != nil {
			app.err.Printf(lm.FailedDeleteOrDisableExpiredUser, user.ID, err)
			continue
		}

		// Sanity check
		if activity.Type != ActivityUnknown {
			app.storage.SetActivityKey(shortuuid.New(), activity, nil, false)
		}

		// Outbound webhook for downstream automation (n8n, Home Assistant, etc.)
		app.fireWebhook("user_expired", map[string]any{
			"user_id":  user.ID,
			"username": user.Name,
			"action":   map[bool]string{true: "deleted", false: "disabled"}[activity.Type == ActivityDeletion],
		})

		// If we're not gonna be deleting the user later, we don't need the expiry stored anymore:
		// 1. Delete after N days is disabled, or Expiry mode is set to delete.
		// 2. User has expired and been deleted after N days.
		// 3. User has expired, but their account is not disabled (i.e. an Admin intervened).
		if deleteAfterPeriod <= 0 || alreadyExpiredShouldDelete || (alreadyExpired && !user.Policy.IsDisabled) {
			app.storage.DeleteUserExpiryKey(user.ID)
		} else if deleteAfterPeriod > 0 && !alreadyExpired {
			// Otherwise, mark the expiry as done pending a delete after N days.
			expiry.DeleteAfterPeriod = true
			// Sure, we haven't contacted them yet, but we're about to
			if shouldContact {
				expiry.LastNotified = time.Now()
			}
			app.storage.SetUserExpiryKey(user.ID, expiry)
		}

		shouldInvalidateCache = true

		if shouldContact {
			name := app.getAddressOrName(user.ID)
			// Skip blank contact info
			if name == "" {
				continue
			}
			msg, err := app.email.constructUserExpired(user.Name, false)
			if err != nil {
				app.err.Printf(lm.FailedConstructExpiryMessage, user.ID, err)
			} else if err := app.sendByID(msg, user.ID); err != nil {
				app.err.Printf(lm.FailedSendExpiryMessage, user.ID, name, err)
			} else {
				app.info.Printf(lm.SentExpiryMessage, user.ID, name)
			}
		}
	}

	if shouldInvalidateCache {
		app.InvalidateJellyfinCache()
	}
}
