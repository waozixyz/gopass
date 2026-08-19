package main

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/waozixyz/kryon/go/kryui"
	password "github.com/waozixyz/pass"
)

const clipboardLifetime = 20 * time.Second

type appView int

const (
	viewGenerate appView = iota
	viewProfiles
	viewSettings
)

//go:embed icon.png
var appIconPNG []byte

//go:embed assets/emoji.ttf
var emojiFontTTF []byte

type app struct {
	site, login, master, exclude    *kryui.TextFieldState
	profileName                     *kryui.TextFieldState
	length, counter                 int32
	lower, upper, digits, symbols   bool
	reveal                          bool
	generated, message, fingerprint string
	clipboard                       clipboardLease
	clearAfter                      int32
	view                            appView
	settings                        guiSettings
	config                          guiConfig
	profiles                        []profileEntry
	selectedProfile                 int
}

func newApp() *app {
	cfg, err := loadGUIConfig()
	a := &app{
		site: kryui.NewTextField(101, 256), login: kryui.NewTextField(102, 256),
		master: kryui.NewPasswordField(103, 1024), exclude: kryui.NewTextField(104, 128),
		profileName: kryui.NewTextField(105, 64),
		length:      16, counter: 1, lower: true, upper: true, digits: true, symbols: true,
		clipboard:       clipboardLease{read: kryui.GetClipboardText, write: kryui.SetClipboardText, now: time.Now},
		settings:        cfg.Settings,
		config:          cfg,
		profiles:        sortedProfiles(cfg.Profiles),
		clearAfter:      int32(cfg.Settings.ClearAfter),
		view:            viewGenerate,
		selectedProfile: -1,
	}
	if err != nil {
		a.message = err.Error()
	}
	return a
}

func main() {
	kryui.SetConfigFlags(kryui.FlagWindowResizable)
	kryui.InitWindow(720, 740, "pass")
	icon := kryui.LoadImageFromMemory(".png", appIconPNG)
	kryui.SetWindowIcon(icon)
	kryui.UnloadImage(icon)
	kryui.SetThemeStyle(kryui.ThemeStyleMaterial)
	kryui.SetCurrentTheme(11, true) // Cobalt dark (fallback palette)
	// Follow the system palette by default; Cobalt dark is the fallback when
	// no system theme can be read.
	kryui.SetThemeSource(kryui.ThemeSourceSystem)
	kryui.SetThemeMode(kryui.ThemeModeSystem)
	kryui.EnsureUIDefaultFont()
	// The emoji fingerprint font serves only the master-password emoji
	// codepoints; all other glyphs keep coming from the theme font.
	kryui.RegisterUIFixedFontData("pass-emoji", ".ttf", emojiFontTTF, password.MasterEmojiCodepoints())
	kryui.SetWindowMinSize(560, 620)
	kryui.EnableEventWaiting()

	a := newApp()
	updateCheckStart()
	for !kryui.WindowShouldClose() && !updateQuitRequested() {
		a.clipboard.tick()
		updatePoll()
		kryui.BeginDrawing()
		w := kryui.GetScreenWidth()
		kryui.BeginUI(kryui.Key("pass/main"))
		a.draw(w)
		kryui.EndUI()
		a.handleUIEvents()
		kryui.EndDrawing()
	}
	a.master.Clear()
	a.clipboard.clear()
	kryui.CloseWindow()
	// With the window down and secrets cleared, a pending update re-execs
	// the staged AppImage here.
	updateExecAfterUI()
}

func (a *app) draw(width int32) {
	bg := kryui.GetThemeBackground()
	scheme := kryui.GetUIMaterialScheme()
	height := kryui.GetScreenHeight()
	kryui.ClearBackground(bg)
	contentW := width - 64
	if contentW > 720 {
		contentW = 720
	}
	x := (width - contentW) / 2
	cardH := height - 104
	if cardH < 520 {
		cardH = 520
	}
	kryui.DrawRectangleRounded(kryui.NewRectangle(float32(x), 22, float32(contentW), float32(cardH)), .035, 10, scheme.Surface)
	kryui.DrawRectangleLinesEx(kryui.NewRectangle(float32(x), 22, float32(contentW), float32(cardH)), 1, scheme.Outline)

	style := inputStyle()
	y := int32(46)
	switch a.view {
	case viewProfiles:
		a.drawProfiles(x+24, y, contentW-48, style)
	case viewSettings:
		a.drawSettings(x+24, y, contentW-48)
	default:
		a.drawGenerate(x+24, y, contentW-48, style)
	}

	a.drawBottomNav(width, height)
	a.updateDrawRow(x+24, 22+cardH+18, contentW-48)
}

func (a *app) drawBottomNav(width, height int32) {
	result := kryui.BottomNav(kryui.BottomNavProps{
		ViewWidth:  width,
		ViewHeight: height,
		Items: []kryui.BottomNavItem{
			{Route: int32(viewGenerate), Label: "Generate", Active: a.view == viewGenerate},
			{Route: int32(viewProfiles), Label: "Profiles", Active: a.view == viewProfiles},
			{Route: int32(viewSettings), Label: "Settings", Active: a.view == viewSettings},
		},
	})
	if result.ClickedRoute >= int32(viewGenerate) && result.ClickedRoute <= int32(viewSettings) {
		a.view = appView(result.ClickedRoute)
	}
}

func (a *app) drawGenerate(x, y, width int32, style kryui.UITextInputStyle) {
	text := kryui.GetThemeText()
	scheme := kryui.GetUIMaterialScheme()
	kryui.DrawUIText("PASSWORD DETAILS", x, y, kryui.UIText12, scheme.Primary)
	y += 30
	a.field("Site", a.site, x, y, width, style)
	y += 76
	a.field("Login", a.login, x, y, width, style)
	y += 76
	kryui.DrawUIText("Master password", x, y, kryui.UIText14, text)
	a.master.SetSecure(!a.reveal)
	kryui.TextField(a.master, kryui.FieldProps{Bounds: kryui.NewRectangle(float32(x), float32(y+24), float32(width-126), 38), Font: kryui.UIText16, Style: style})
	if a.settings.ShowFingerprint && a.fingerprint != "" {
		fontToken := kryui.PushUIFont("pass-emoji")
		fpWidth := kryui.MeasureText(a.fingerprint, kryui.UIText16)
		kryui.DrawUIText(a.fingerprint, x+width-fpWidth, y, kryui.UIText16, scheme.OnSurfaceVariant)
		kryui.PopUIFont(fontToken)
	}
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+width-88), float32(y+24), 88, 38), Label: map[bool]string{true: "Hide", false: "Reveal"}[a.reveal], Style: kryui.UIButtonStyleSecondary, ID: 205})
	y += 76

	kryui.DrawUIText("PASSWORD RULES", x, y-10, kryui.UIText12, scheme.Primary)
	y += 14
	kryui.DrawUIText("Length", x, y, kryui.UIText14, text)
	kryui.Spinbox(kryui.SpinboxProps{Bounds: kryui.NewRectangle(float32(x), float32(y+24), 130, 38), ID: 301, Min: 1, Max: 128, Step: 1, Value: &a.length})
	kryui.DrawUIText("Counter", x+158, y, kryui.UIText14, text)
	kryui.Spinbox(kryui.SpinboxProps{Bounds: kryui.NewRectangle(float32(x+158), float32(y+24), 130, 38), ID: 302, Min: 1, Max: 999999, Step: 1, Value: &a.counter})
	kryui.TextField(a.exclude, kryui.FieldProps{Bounds: kryui.NewRectangle(float32(x+316), float32(y+24), float32(width-316), 38), Font: kryui.UIText16, Style: style})
	kryui.DrawUIText("Excluded characters", x+316, y, kryui.UIText14, text)
	y += 76

	kryui.DrawUICheckboxToggle(x, y, "Lowercase", &a.lower)
	kryui.DrawUICheckboxToggle(x+150, y, "Uppercase", &a.upper)
	kryui.DrawUICheckboxToggle(x+300, y, "Digits", &a.digits)
	kryui.DrawUICheckboxToggle(x+420, y, "Symbols", &a.symbols)
	y += 48
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x), float32(y), 150, 42), Label: "Generate", Style: kryui.UIButtonStylePrimary, ID: 401})
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+166), float32(y), 150, 42), Label: "Copy", Style: kryui.UIButtonStyleSecondary, ID: 402, Disabled: a.generated == ""})
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+332), float32(y), 120, 42), Label: "Clear", Style: kryui.UIButtonStyleSecondary, ID: 403})
	y += 56
	kryui.DrawRectangleRounded(kryui.NewRectangle(float32(x), float32(y), float32(width), 72), .08, 10, scheme.SurfaceContainer)
	if a.generated != "" {
		kryui.SelectableText(a.generated, x+18, y+15, kryui.UIText20, text)
	} else {
		kryui.DrawUIText("Your generated password appears here", x+18, y+17, kryui.UIText16, scheme.OnSurfaceVariant)
	}
	if a.message != "" {
		kryui.DrawUIText(a.message, x+18, y+45, kryui.UIText12, scheme.OnSurfaceVariant)
	}
}

func (a *app) drawProfiles(x, y, width int32, style kryui.UITextInputStyle) {
	text := kryui.GetThemeText()
	scheme := kryui.GetUIMaterialScheme()

	kryui.DrawUIText("PROFILE", x, y, kryui.UIText12, scheme.Primary)
	y += 30
	a.field("Name", a.profileName, x, y, width, style)
	y += 76
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x), float32(y), 154, 42), Label: "Save Current", Style: kryui.UIButtonStylePrimary, ID: 501})
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+170), float32(y), 120, 42), Label: "Delete", Style: kryui.UIButtonStyleSecondary, ID: 502, Disabled: a.selectedProfile < 0})
	y += 62

	kryui.DrawUIText("SAVED PROFILES", x, y, kryui.UIText12, scheme.Primary)
	y += 28
	if len(a.profiles) == 0 {
		kryui.DrawUIText("No profiles saved yet", x, y, kryui.UIText14, scheme.OnSurfaceVariant)
		y += 34
	}
	for i, p := range a.profiles {
		if i >= 7 {
			kryui.DrawUIText(fmt.Sprintf("%d more profiles in settings file", len(a.profiles)-i), x, y+8, kryui.UIText12, scheme.OnSurfaceVariant)
			y += 32
			break
		}
		label := p.Name
		if p.Data.Site != "" {
			label = fmt.Sprintf("%s  -  %s", p.Name, p.Data.Site)
		}
		style := kryui.UIButtonStyleSecondary
		if i == a.selectedProfile {
			style = kryui.UIButtonStylePrimary
		}
		kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x), float32(y), float32(width), 40), Label: label, Style: style, ID: int32(600 + i)})
		y += 48
	}
	if a.message != "" {
		kryui.DrawUIText(a.message, x, y+8, kryui.UIText12, text)
	}
}

func (a *app) drawSettings(x, y, width int32) {
	text := kryui.GetThemeText()
	scheme := kryui.GetUIMaterialScheme()

	kryui.DrawUIText("BEHAVIOR", x, y, kryui.UIText12, scheme.Primary)
	y += 32
	kryui.DrawUICheckboxToggle(x, y, "Auto-copy after Generate", &a.settings.AutoCopy)
	y += 42
	kryui.DrawUICheckboxToggle(x, y, "Show master fingerprint", &a.settings.ShowFingerprint)
	y += 48
	kryui.DrawUIText("Clipboard seconds", x, y, kryui.UIText14, text)
	kryui.Spinbox(kryui.SpinboxProps{Bounds: kryui.NewRectangle(float32(x), float32(y+24), 150, 38), ID: 701, Min: 0, Max: 3600, Step: 5, Value: &a.clearAfter})
	y += 82

	kryui.DrawUIText("MASTER PASSWORD", x, y, kryui.UIText12, scheme.Primary)
	y += 30
	kryui.DrawUIText("Android can save and unlock the master password with system biometrics.", x, y, kryui.UIText14, scheme.OnSurfaceVariant)
	y += 34
	kryui.DrawUIText("Desktop keeps the master password only in memory for this session.", x, y, kryui.UIText14, scheme.OnSurfaceVariant)
	y += 54
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x), float32(y), 140, 42), Label: "Save Settings", Style: kryui.UIButtonStylePrimary, ID: 503})
	y += 58
	if a.message != "" {
		kryui.DrawRectangleRounded(kryui.NewRectangle(float32(x), float32(y), float32(width), 56), .08, 10, scheme.SurfaceContainer)
		kryui.DrawUIText(a.message, x+16, y+20, kryui.UIText12, scheme.OnSurfaceVariant)
	}
}

func (a *app) field(label string, field *kryui.TextFieldState, x, y, width int32, style kryui.UITextInputStyle) {
	kryui.DrawUIText(label, x, y, kryui.UIText14, kryui.GetThemeText())
	kryui.TextField(field, kryui.FieldProps{Bounds: kryui.NewRectangle(float32(x), float32(y+24), float32(width), 38), Font: kryui.UIText16, Style: style})
}

func (a *app) handleUIEvents() {
	for event, ok := kryui.NextUIEvent(); ok; event, ok = kryui.NextUIEvent() {
		switch event.Kind {
		case kryui.EventTextChanged:
			if event.Key == 103 {
				if master := a.master.Text(); master != "" {
					a.fingerprint = password.MasterPasswordEmojiString(master)
				} else {
					a.fingerprint = ""
				}
			}
		case kryui.EventClick:
			switch event.Key {
			case 205:
				a.reveal = !a.reveal
			case 401:
				a.generate()
			case 402:
				if a.generated != "" {
					a.copyGenerated()
				}
			case 403:
				a.master.Clear()
				a.fingerprint = ""
				a.generated = ""
				a.message = "Cleared"
			case 501:
				a.saveCurrentProfile()
			case 502:
				a.deleteSelectedProfile()
			case 503:
				a.saveSettings()
			default:
				if event.Key >= 600 && event.Key < kryui.UIKey(600+len(a.profiles)) {
					idx := int(event.Key - 600)
					a.selectedProfile = idx
					a.applyProfile(a.profiles[idx])
					a.view = viewGenerate
				}
			}
		}
	}
}

func (a *app) generate() {
	result, err := password.Generate(a.site.Text(), a.login.Text(), a.master.Text(), password.Options{Length: int(a.length), Counter: uint64(a.counter), Lowercase: a.lower, Uppercase: a.upper, Digits: a.digits, Symbols: a.symbols, Exclude: a.exclude.Text()})
	if err != nil {
		a.generated = ""
		a.message = err.Error()
		return
	}
	a.generated = result
	if a.settings.AutoCopy {
		a.copyGenerated()
		return
	}
	a.message = "Password generated locally"
}

func (a *app) copyGenerated() {
	seconds := int(a.clearAfter)
	if seconds < 0 {
		seconds = 0
	}
	a.clipboard.copy(a.generated, clipboardDuration(seconds))
	if seconds > 0 {
		a.message = fmt.Sprintf("Copied; clipboard clears in %d seconds", seconds)
	} else {
		a.message = "Copied"
	}
}

func clipboardDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func (a *app) saveSettings() {
	if a.clearAfter < 0 {
		a.clearAfter = 0
	}
	a.settings.ClearAfter = int(a.clearAfter)
	a.config.Settings = a.settings
	if err := saveGUIConfig(a.config); err != nil {
		a.message = err.Error()
		return
	}
	a.message = "Settings saved"
}

func (a *app) saveCurrentProfile() {
	name := strings.TrimSpace(a.profileName.Text())
	if name == "" {
		a.message = "Enter a profile name"
		return
	}
	if a.config.Profiles == nil {
		a.config.Profiles = map[string]storedProfile{}
	}
	a.config.Profiles[name] = storedProfile{
		Site:      a.site.Text(),
		Login:     a.login.Text(),
		Counter:   uint64(a.counter),
		Length:    int(a.length),
		Lowercase: a.lower,
		Uppercase: a.upper,
		Digits:    a.digits,
		Symbols:   a.symbols,
		Exclude:   a.exclude.Text(),
	}
	a.config.Settings = a.settings
	if err := saveGUIConfig(a.config); err != nil {
		a.message = err.Error()
		return
	}
	a.profiles = sortedProfiles(a.config.Profiles)
	a.selectedProfile = -1
	for i, p := range a.profiles {
		if p.Name == name {
			a.selectedProfile = i
			break
		}
	}
	a.message = "Profile saved"
}

func (a *app) deleteSelectedProfile() {
	if a.selectedProfile < 0 || a.selectedProfile >= len(a.profiles) {
		return
	}
	name := a.profiles[a.selectedProfile].Name
	delete(a.config.Profiles, name)
	if err := saveGUIConfig(a.config); err != nil {
		a.message = err.Error()
		return
	}
	a.profiles = sortedProfiles(a.config.Profiles)
	a.selectedProfile = -1
	a.profileName.SetText("")
	a.message = "Profile deleted"
}

func (a *app) applyProfile(p profileEntry) {
	a.profileName.SetText(p.Name)
	a.site.SetText(p.Data.Site)
	a.login.SetText(p.Data.Login)
	if p.Data.Counter > 0 {
		a.counter = int32(p.Data.Counter)
	}
	if p.Data.Length > 0 {
		a.length = int32(p.Data.Length)
	}
	a.lower = p.Data.Lowercase
	a.upper = p.Data.Uppercase
	a.digits = p.Data.Digits
	a.symbols = p.Data.Symbols
	a.exclude.SetText(p.Data.Exclude)
	a.message = "Profile loaded"
}

func inputStyle() kryui.UITextInputStyle {
	s := kryui.GetUIMaterialScheme()
	return kryui.UITextInputStyle{Background: s.SurfaceContainer, Border: s.Outline, FocusBorder: s.Primary, Text: s.OnSurface, Cursor: s.Primary, Radius: 6, PaddingX: 10, PaddingY: 8}
}
