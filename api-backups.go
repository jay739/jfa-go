package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"
	lm "github.com/jay739/omnifin/logmessages"
)

// @Summary Creates a backup of the database.
// @Router /backups [post]
// @Success 200 {object} CreateBackupDTO
// @Security Bearer
// @tags Backups
func (app *appContext) CreateBackup(gc *gin.Context) {
	backup := app.makeBackup()
	gc.JSON(200, backup)
}

// @Summary Download a specific backup file. Requires auth, so can't be accessed plainly in the browser.
// @Param fname path string true "backup filename"
// @Router /backups/{fname} [get]
// @Produce octet-stream
// @Produce json
// @Success 200 {body} file
// @Failure 400 {object} boolResponse
// @Security Bearer
// @tags Backups
func (app *appContext) GetBackup(gc *gin.Context) {
	escapedFName := gc.Param("fname")
	fname, err := url.QueryUnescape(escapedFName)
	if err != nil {
		respondBool(400, false, gc)
		return
	}
	// Defence in depth against path traversal: strip any directory components
	// before we ever join against the backups dir. Belt-and-suspenders alongside
	// Backup.FromString's format check below.
	fname = filepath.Base(fname)
	b := Backup{}
	err = b.FromString(fname)
	if err != nil || b.Date.IsZero() {
		app.debug.Printf(lm.IgnoreInvalidFilename, fname, err)
		respondBool(400, false, gc)
		return
	}
	path := app.config.Section("backups").Key("path").String()
	fullpath := filepath.Join(path, fname)
	gc.FileAttachment(fullpath, fname)
}

// @Summary Get a list of backups.
// @Router /backups [get]
// @Produce json
// @Success 200 {object} GetBackupsDTO
// @Security Bearer
// @tags Backups
func (app *appContext) GetBackups(gc *gin.Context) {
	path := app.config.Section("backups").Key("path").String()
	backups := app.getBackups()
	sort.Sort(backups)
	resp := GetBackupsDTO{}
	resp.Backups = make([]CreateBackupDTO, backups.count)

	for i, item := range backups.files[:backups.count] {
		resp.Backups[i].Name = item.Name()
		fullpath := filepath.Join(path, item.Name())
		resp.Backups[i].Path = fullpath
		resp.Backups[i].Date = backups.info[i].Date.Unix()
		resp.Backups[i].Commit = backups.info[i].Commit
		fstat, err := os.Stat(fullpath)
		if err == nil {
			resp.Backups[i].Size = fileSize(fstat.Size())
		}
	}
	gc.JSON(200, resp)
}

// @Summary Restore a backup file stored locally to the server.
// @Param fname path string true "backup filename"
// @Router /backups/restore/{fname} [post]
// @Produce json
// @Failure 400 {object} boolResponse
// @Security Bearer
// @tags Backups
func (app *appContext) RestoreLocalBackup(gc *gin.Context) {
	fname := gc.Param("fname")
	// Strip any directory components before validation so an attacker can't
	// reach files outside the backups directory even if they bypass the format
	// check below.
	fname = filepath.Base(fname)
	b := Backup{}
	err := b.FromString(fname)
	if err != nil || b.Date.IsZero() {
		app.debug.Printf(lm.IgnoreInvalidFilename, fname, err)
		respondBool(400, false, gc)
		return
	}
	path := app.config.Section("backups").Key("path").String()
	fullpath := filepath.Join(path, fname)
	LOADBAK = fullpath
	app.restart(gc)
}

// @Summary Restore a backup file uploaded by the user.
// @Param file formData file true ".bak file"
// @Router /backups/restore [post]
// @Produce json
// @Failure 400 {object} boolResponse
// @Security Bearer
// @tags Backups
func (app *appContext) RestoreBackup(gc *gin.Context) {
	const maxBackupBytes = 100 << 20 // 100 MB
	gc.Request.Body = http.MaxBytesReader(gc.Writer, gc.Request.Body, maxBackupBytes)
	file, err := gc.FormFile("backups-file")
	if err != nil {
		app.err.Printf(lm.FailedGetUpload, err)
		respondBool(400, false, gc)
		return
	}
	if file.Size > maxBackupBytes {
		respondBool(400, false, gc)
		return
	}
	app.debug.Printf(lm.GetUpload, file.Filename)
	path := app.config.Section("backups").Key("path").String()
	b := Backup{Upload: true}
	fullpath := filepath.Join(path, b.String())
	gc.SaveUploadedFile(file, fullpath)
	app.debug.Printf(lm.Write, fullpath)
	LOADBAK = fullpath
	app.restart(gc)
}
