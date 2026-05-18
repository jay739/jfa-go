package main

import (
	"net/http"
	"net/url"
	"time"

	"github.com/jay739/omnifin/common"
	"github.com/jay739/omnifin/logger"
	lm "github.com/jay739/omnifin/logmessages"
)

type WebhookSender struct {
	httpClient     *http.Client
	timeoutHandler common.TimeoutHandler
	log            *logger.Logger
}

// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
func (ws *WebhookSender) SetTransport(t *http.Transport) {
	ws.httpClient.Transport = t
}

func NewWebhookSender(timeoutHandler common.TimeoutHandler, log *logger.Logger) *WebhookSender {
	return &WebhookSender{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeoutHandler: timeoutHandler,
		log:            log,
	}
}

func (ws *WebhookSender) Send(uri string, payload any) (int, error) {
	_, status, err := common.Req(ws.httpClient, ws.timeoutHandler, http.MethodPost, uri, payload, url.Values{}, nil, true)
	ws.log.Printf(lm.WebhookRequest, uri, status, err)
	return status, err
}

// fireWebhook reads `[webhooks].<event>` from the config (one or more pipe-separated URIs)
// and POSTs the given payload to each, asynchronously and best-effort. Failures are logged
// but do not block the caller.
//
// Supported events (any [webhooks].<key> in config.ini becomes an emission target):
//   - created          (existing) user successfully created via invite
//   - user_disabled    user disabled by admin or by expiry
//   - user_enabled     user re-enabled
//   - user_deleted     user removed
//   - user_expired     expiry daemon disabled or deleted a user
//   - expiry_extended  expiry daemon auto-extended a user's expiry
//   - invite_used      a user successfully completed signup via invite
//   - announcement_sent admin sent an announcement to one or more users
// webhookSemaphore caps the number of in-flight outbound webhook requests so a
// poorly-configured event burst (or a malicious config with many URIs) can't
// exhaust goroutines or sockets. Sized for typical homelab usage; tune if
// you wire dozens of integrations.
var webhookSemaphore = make(chan struct{}, 16)

func (app *appContext) fireWebhook(event string, payload any) {
	uris := app.config.Section("webhooks").Key(event).StringsWithShadows("|")
	if len(uris) == 0 {
		return
	}
	body := map[string]any{
		"event":   event,
		"payload": payload,
		"sent_at": time.Now().UTC().Format(time.RFC3339),
	}
	for _, uri := range uris {
		uri := uri
		go func() {
			webhookSemaphore <- struct{}{}
			defer func() { <-webhookSemaphore }()
			if _, err := app.webhooks.Send(uri, body); err != nil {
				app.debug.Printf("webhook %s -> %s failed: %v", event, uri, err)
			}
		}()
	}
}
