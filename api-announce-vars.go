package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// announceVarsDTO is returned by /users/announce-vars and provides ready-to-substitute
// markdown values for the announcement editor.
type announceVarsDTO struct {
	Vars map[string]string `json:"vars"`
}

// jfLatestItem is a minimal projection of Jellyfin /Items results used for variable rendering.
type jfLatestItem struct {
	Id              string   `json:"Id"`
	Name            string   `json:"Name"`
	Overview        string   `json:"Overview,omitempty"`
	ProductionYear  int      `json:"ProductionYear,omitempty"`
	OfficialRating  string   `json:"OfficialRating,omitempty"`
	CommunityRating float64  `json:"CommunityRating,omitempty"`
	Type            string   `json:"Type,omitempty"`
	Genres          []string `json:"Genres,omitempty"`
	RunTimeTicks    int64    `json:"RunTimeTicks,omitempty"`
}

type jfLatestResp struct {
	Items []jfLatestItem `json:"Items"`
}

type announceStatsResp struct {
	Vars map[string]string `json:"vars"`
}

// jfFetchLatest queries Jellyfin's /Items endpoint for items of the given type, sorted by date added.
func (app *appContext) jfFetchLatest(itemType string, limit int) ([]jfLatestItem, error) {
	q := url.Values{}
	q.Set("Recursive", "true")
	q.Set("IncludeItemTypes", itemType)
	q.Set("SortBy", "DateCreated")
	q.Set("SortOrder", "Descending")
	q.Set("Limit", fmt.Sprintf("%d", limit))
	q.Set("Fields", "ProductionYear,OfficialRating,CommunityRating,Genres,RunTimeTicks,Overview")
	endpoint := fmt.Sprintf("%s/Items?%s", strings.TrimRight(app.jf.Server, "/"), q.Encode())

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Emby-Token", app.jf.AccessToken)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin returned %d: %s", resp.StatusCode, string(body))
	}
	var parsed jfLatestResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Items, nil
}

// renderItemList formats Jellyfin items as a markdown bullet list.
func renderItemList(items []jfLatestItem) string {
	if len(items) == 0 {
		return "_(none yet)_"
	}
	var b strings.Builder
	for _, item := range items {
		b.WriteString("- **")
		b.WriteString(item.Name)
		b.WriteString("**")
		if item.ProductionYear > 0 {
			b.WriteString(fmt.Sprintf(" (%d)", item.ProductionYear))
		}
		if item.CommunityRating > 0 {
			b.WriteString(fmt.Sprintf(" — ⭐ %.1f", item.CommunityRating))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderTopGenres returns a markdown bullet list of the most common genres across the given items.
func renderTopGenres(items []jfLatestItem, limit int) string {
	counts := map[string]int{}
	for _, it := range items {
		for _, g := range it.Genres {
			if g == "" {
				continue
			}
			counts[g]++
		}
	}
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	// Simple sort: bubble (small N).
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].v > pairs[i].v {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	if len(pairs) == 0 {
		return "_(no genre data)_"
	}
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(fmt.Sprintf("- **%s** (%d)\n", p.k, p.v))
	}
	return strings.TrimRight(b.String(), "\n")
}

// pickLongest returns the title of the longest-runtime item, formatted as "Title (Hh Mm)".
func pickLongest(items []jfLatestItem) string {
	if len(items) == 0 {
		return ""
	}
	best := items[0]
	for _, it := range items[1:] {
		if it.RunTimeTicks > best.RunTimeTicks {
			best = it
		}
	}
	if best.RunTimeTicks <= 0 {
		return best.Name
	}
	// Jellyfin uses 100ns ticks; convert to minutes.
	mins := best.RunTimeTicks / 10_000_000 / 60
	h, m := mins/60, mins%60
	if h > 0 {
		return fmt.Sprintf("%s (%dh %dm)", best.Name, h, m)
	}
	return fmt.Sprintf("%s (%dm)", best.Name, m)
}

// renderItemGrid generates an email-safe HTML table grid of poster images with titles.
// cols controls how many posters per row; limit caps the total items shown.
func renderItemGrid(items []jfLatestItem, publicServer string, cols, limit int) string {
	if len(items) == 0 || publicServer == "" {
		return renderItemList(items)
	}
	srv := strings.TrimRight(publicServer, "/")
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	var b strings.Builder
	b.WriteString(`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="width:100%;border-collapse:collapse;">`)
	for i, item := range items {
		if i%cols == 0 {
			b.WriteString(`<tr>`)
		}
		imgURL := fmt.Sprintf("%s/Items/%s/Images/Primary?maxWidth=200&quality=85", srv, html.EscapeString(item.Id))
		detailURL := fmt.Sprintf("%s/web/index.html#!/details?id=%s", srv, html.EscapeString(item.Id))
		title := html.EscapeString(item.Name)
		var meta strings.Builder
		if item.ProductionYear > 0 {
			meta.WriteString(fmt.Sprintf("%d", item.ProductionYear))
		}
		if item.CommunityRating > 0 {
			if meta.Len() > 0 {
				meta.WriteString(" · ")
			}
			meta.WriteString(fmt.Sprintf("⭐ %.1f", item.CommunityRating))
		}
		b.WriteString(fmt.Sprintf(
			`<td style="width:%d%%;vertical-align:top;padding:6px;" valign="top">
			<a href="%s" style="text-decoration:none;color:inherit;">
				<img src="%s" alt="%s" width="100%%" style="display:block;width:100%%;border-radius:6px;aspect-ratio:2/3;object-fit:cover;background:#1f2937;" />
				<div style="margin-top:6px;font-size:12px;font-weight:600;color:#f9fafb;line-height:1.3;">%s</div>`,
			100/cols, detailURL, imgURL, title, title,
		))
		if meta.Len() > 0 {
			b.WriteString(fmt.Sprintf(`<div style="font-size:11px;color:#9ca3af;margin-top:2px;">%s</div>`, html.EscapeString(meta.String())))
		}
		b.WriteString(`</a></td>`)
		if i%cols == cols-1 || i == len(items)-1 {
			// Pad incomplete row.
			for pad := (i % cols) + 1; pad < cols; pad++ {
				b.WriteString(fmt.Sprintf(`<td style="width:%d%%;">&nbsp;</td>`, 100/cols))
			}
			b.WriteString(`</tr>`)
		}
	}
	b.WriteString(`</table>`)
	return b.String()
}

// renderFeaturedCard generates a styled HTML card for a single item with poster, metadata, and a watch link.
func renderFeaturedCard(item jfLatestItem, publicServer string) string {
	if item.Id == "" || publicServer == "" {
		return fmt.Sprintf("**%s**", item.Name)
	}
	srv := strings.TrimRight(publicServer, "/")
	imgURL := fmt.Sprintf("%s/Items/%s/Images/Primary?maxWidth=300&quality=90", srv, html.EscapeString(item.Id))
	detailURL := fmt.Sprintf("%s/web/index.html#!/details?id=%s", srv, html.EscapeString(item.Id))
	title := html.EscapeString(item.Name)

	var metaParts []string
	if item.ProductionYear > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d", item.ProductionYear))
	}
	if item.OfficialRating != "" {
		metaParts = append(metaParts, html.EscapeString(item.OfficialRating))
	}
	if item.CommunityRating > 0 {
		metaParts = append(metaParts, fmt.Sprintf("⭐ %.1f", item.CommunityRating))
	}
	metaLine := strings.Join(metaParts, " · ")

	var genreHTML string
	if len(item.Genres) > 0 {
		limit := item.Genres
		if len(limit) > 3 {
			limit = limit[:3]
		}
		var gs []string
		for _, g := range limit {
			gs = append(gs, fmt.Sprintf(`<span style="display:inline-block;background:rgba(99,102,241,0.15);border-radius:4px;padding:2px 7px;font-size:11px;color:#a5b4fc;margin-right:4px;">%s</span>`, html.EscapeString(g)))
		}
		genreHTML = strings.Join(gs, "")
	}

	var overviewHTML string
	if item.Overview != "" {
		ov := item.Overview
		if len(ov) > 220 {
			ov = ov[:220] + "…"
		}
		overviewHTML = fmt.Sprintf(`<p style="font-size:13px;line-height:1.6;color:#9ca3af;margin:10px 0 0;">%s</p>`, html.EscapeString(ov))
	}

	return fmt.Sprintf(`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="width:100%%;border-collapse:collapse;background:rgba(255,255,255,0.03);border-radius:10px;overflow:hidden;">
<tr>
<td style="width:120px;vertical-align:top;padding:0;" valign="top">
<a href="%s"><img src="%s" alt="%s" width="120" style="display:block;width:120px;border-radius:8px 0 0 8px;aspect-ratio:2/3;object-fit:cover;background:#1f2937;" /></a>
</td>
<td style="vertical-align:top;padding:14px 16px;" valign="top">
<div style="font-size:17px;font-weight:700;color:#f9fafb;line-height:1.3;margin-bottom:6px;"><a href="%s" style="text-decoration:none;color:#f9fafb;">%s</a></div>
<div style="font-size:12px;color:#6b7280;margin-bottom:8px;">%s</div>
<div style="margin-bottom:8px;">%s</div>
%s
<div style="margin-top:12px;"><a href="%s" style="display:inline-block;background:#6366f1;color:#fff;font-size:12px;font-weight:600;padding:6px 14px;border-radius:6px;text-decoration:none;">▶ Watch Now</a></div>
</td>
</tr>
</table>`,
		detailURL, imgURL, title,
		detailURL, title,
		metaLine,
		genreHTML,
		overviewHTML,
		detailURL,
	)
}

// @Summary Get available variables for announcement template substitution.
// @Produce json
// @Success 200 {object} announceVarsDTO
// @Router /users/announce-vars [get]
// @Security Bearer
// @tags Users
func (app *appContext) GetAnnounceVars(gc *gin.Context) {
	vars := buildAnnounceVars(app)
	gc.JSON(200, announceVarsDTO{Vars: vars})
}

// buildAnnounceVars centralizes variable construction so it can be reused by the file-load endpoint.
func buildAnnounceVars(app *appContext) map[string]string {
	vars := map[string]string{}

	now := time.Now()
	vars["date"] = now.Format("January 2, 2006")
	vars["month_year"] = now.Format("January 2006")
	vars["weekday"] = now.Format("Monday")

	if pubURL := app.config.Section("jellyfin").Key("public_server").MustString(""); pubURL != "" {
		vars["server_url"] = pubURL
	} else if srv := app.config.Section("jellyfin").Key("server").MustString(""); srv != "" {
		vars["server_url"] = srv
	}
	if app.jf.ServerInfo.Name != "" {
		vars["server_name"] = app.jf.ServerInfo.Name
	}

	pubServer := vars["server_url"]

	movies, mErr := app.jfFetchLatest("Movie", 10)
	if mErr == nil {
		vars["recent_movies"] = renderItemList(movies)
		vars["recent_movies_grid"] = renderItemGrid(movies, pubServer, 3, 6)
		vars["top_genres"] = renderTopGenres(movies, 5)
		vars["longest_movie"] = pickLongest(movies)
		if len(movies) > 0 {
			vars["featured_movie"] = renderFeaturedCard(movies[0], pubServer)
		}
	} else {
		vars["recent_movies"] = "_(could not fetch movies)_"
		vars["recent_movies_grid"] = "_(could not fetch movies)_"
	}
	if series, err := app.jfFetchLatest("Series", 10); err == nil {
		vars["recent_shows"] = renderItemList(series)
		vars["recent_shows_grid"] = renderItemGrid(series, pubServer, 3, 6)
		if len(series) > 0 {
			vars["featured_show"] = renderFeaturedCard(series[0], pubServer)
		}
	} else {
		vars["recent_shows"] = "_(could not fetch shows)_"
		vars["recent_shows_grid"] = "_(could not fetch shows)_"
	}
	if eps, err := app.jfFetchLatest("Episode", 10); err == nil {
		vars["recent_episodes"] = renderItemList(eps)
	} else {
		vars["recent_episodes"] = "_(could not fetch episodes)_"
	}

	if users, err := app.jf.GetUsers(false); err == nil {
		vars["user_count"] = fmt.Sprintf("%d", len(users))
		// active in last 30 days = users with LastActivityDate within window.
		cutoff := now.Add(-30 * 24 * time.Hour)
		active := 0
		for _, u := range users {
			if !u.LastActivityDate.Time.IsZero() && u.LastActivityDate.Time.After(cutoff) {
				active++
			}
		}
		vars["active_users_30d"] = fmt.Sprintf("%d", active)
	}

	for key, val := range app.fetchAnnouncementStatsVars(30) {
		if val != "" {
			vars[key] = val
		}
	}

	return vars
}

func (app *appContext) fetchAnnouncementStatsVars(days int) map[string]string {
	baseURL := os.Getenv("JELLYFIN_STATS_API_URL")
	if baseURL == "" {
		baseURL = "http://jellyfin-stats-api:3020"
	}
	endpoint := fmt.Sprintf("%s/api/announcement-summary?days=%d", strings.TrimRight(baseURL, "/"), days)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return map[string]string{}
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		app.debug.Printf("Failed to fetch announcement stats from %s: %v", endpoint, err)
		return map[string]string{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		app.debug.Printf("Announcement stats API returned %d: %s", resp.StatusCode, string(body))
		return map[string]string{}
	}
	var parsed announceStatsResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		app.debug.Printf("Failed to decode announcement stats response: %v", err)
		return map[string]string{}
	}
	return parsed.Vars
}

// substituteAnnounceVars replaces {{var}} placeholders in `content` with values from `vars`.
// Unknown placeholders are left intact so the admin can see and fix them.
func substituteAnnounceVars(content string, vars map[string]string) string {
	out := content
	for key, val := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", val)
		// Allow whitespace inside braces: {{ key }}
		out = strings.ReplaceAll(out, "{{ "+key+" }}", val)
	}
	return out
}
