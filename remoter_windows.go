package main

import (
	"github.com/getlantern/systray"

	_ "embed"
	"fmt"
)

//go:embed remoter.ico
var appIcon []byte

func setStatus(format string, args ...any) {
	systray.SetTooltip("Remoter: " + fmt.Sprintf(format, args...))
}

func main() {
	systray.Run(func() {
		systray.SetIcon(appIcon)
		systray.SetTitle("Remoter")
		quit := systray.AddMenuItem("Quit", "Quit Remoter")
		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()

		go func() {
			mainBody()
			systray.Quit()
		}()
	},
		func() {})
}
