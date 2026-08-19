package main

// Desktop update flow over kryon's kry_update_flow: check the release
// appcast once per session, toast when a newer pass exists, and offer
// download + restart in a small row under the main card. AppImage builds
// self-update (verified download, atomic swap, re-exec); tarball, deb and
// source installs stay on the release link.

import (
	"github.com/waozixyz/kryon/go/kryui"
	password "github.com/waozixyz/pass"
)

const appcastURL = "https://github.com/waozixyz/pass/releases/latest/download/appcast.json"

const updateButtonID = 510

var (
	updater        *kryui.UpdateFlow
	updateNoticed  bool
	restartPending bool
)

func updateCheckStart() {
	updater = kryui.StartUpdateFlow("pass", password.Version, appcastURL)
}

func updatePoll() {
	if updater == nil {
		return
	}
	updater.Poll()
	if !updateNoticed && updater.State() == kryui.UpdateFlowAvailable {
		updateNoticed = true
		if updater.HasArtifact() {
			kryui.ShowToast("pass " + updater.NewVersion() + " available — update below")
		}
	}
}

func updateQuitRequested() bool {
	return restartPending
}

// updateExecAfterUI re-execs the staged AppImage; call once the window is
// closed and secrets are cleared. Does nothing when no restart is pending.
func updateExecAfterUI() {
	if updater != nil && restartPending {
		updater.ExecPending()
	}
}

// updateDrawRow renders the update row under the main card. Returns true
// when it drew (the layout reserves the space unconditionally once an
// update was seen, so the card doesn't jump while states change).
func (a *app) updateDrawRow(x, y, width int32) bool {
	if updater == nil {
		return false
	}
	scheme := kryui.GetUIMaterialScheme()
	switch updater.State() {
	case kryui.UpdateFlowAvailable:
		if !updater.HasArtifact() {
			// system-managed or source install: one quiet link-out line
			if updater.ReleaseURL() == "" {
				return false
			}
			kryui.DrawUIText("pass "+updater.NewVersion()+" available", x, y, kryui.UIText12, scheme.OnSurfaceVariant)
			return true
		}
		kryui.DrawUIText("pass "+updater.NewVersion()+" available", x, y, kryui.UIText12, scheme.OnSurfaceVariant)
		if kryui.Button(kryui.ButtonProps{
			Bounds: kryui.NewRectangle(float32(x+width-96), float32(y-14), 96, 32),
			Label:  "Download", Style: kryui.UIButtonStyleSecondary, ID: updateButtonID,
		}) {
			updater.Download()
		}
	case kryui.UpdateFlowDownloading:
		pct := int(updater.Progress()*100 + 0.5)
		if pct < 0 {
			pct = 0
		}
		kryui.DrawUIText("Downloading… "+itoa(pct)+"%", x, y, kryui.UIText12, scheme.OnSurfaceVariant)
	case kryui.UpdateFlowReady:
		kryui.DrawUIText("pass "+updater.NewVersion()+" ready to install", x, y, kryui.UIText12, scheme.OnSurfaceVariant)
		if kryui.Button(kryui.ButtonProps{
			Bounds: kryui.NewRectangle(float32(x+width-150), float32(y-14), 150, 32),
			Label:  "Restart to update", Style: kryui.UIButtonStylePrimary, ID: updateButtonID,
		}) {
			if updater.Apply() {
				restartPending = true
			}
		}
	case kryui.UpdateFlowFailed:
		kryui.DrawUIText("Update failed: "+updater.Error(), x, y, kryui.UIText12, scheme.OnSurfaceVariant)
		if kryui.Button(kryui.ButtonProps{
			Bounds: kryui.NewRectangle(float32(x+width-96), float32(y-14), 96, 32),
			Label:  "Retry", Style: kryui.UIButtonStyleSecondary, ID: updateButtonID,
		}) {
			updater.Download()
		}
	default:
		return false
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
