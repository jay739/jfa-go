package main

import (
	"fmt"

	lm "github.com/jay739/omnifin/logmessages"
)

func (app *appContext) HardRestart() error {
	return fmt.Errorf(lm.FailedHardRestartWindows)
}
