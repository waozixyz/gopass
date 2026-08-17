package main

import (
	_ "embed"
	"time"

	password "github.com/waozixyz/gopass"
	"github.com/waozixyz/kryon/go/kryui"
)

const clipboardLifetime = 20 * time.Second

//go:embed icon.png
var appIconPNG []byte

//go:embed assets/emoji.ttf
var emojiFontTTF []byte

type app struct {
	site, login, master, exclude    *kryui.TextFieldState
	length, counter                 int32
	lower, upper, digits, symbols   bool
	reveal                          bool
	generated, message, fingerprint string
	clipboard                       clipboardLease
}

func newApp() *app {
	return &app{
		site: kryui.NewTextField(101, 256), login: kryui.NewTextField(102, 256),
		master: kryui.NewPasswordField(103, 1024), exclude: kryui.NewTextField(104, 128),
		length: 16, counter: 1, lower: true, upper: true, digits: true, symbols: true,
		clipboard: clipboardLease{read: kryui.GetClipboardText, write: kryui.SetClipboardText, now: time.Now},
	}
}

func main() {
	kryui.SetConfigFlags(kryui.FlagWindowResizable)
	kryui.InitWindow(720, 690, "gopass")
	defer kryui.CloseWindow()
	icon := kryui.LoadImageFromMemory(".png", appIconPNG)
	kryui.SetWindowIcon(icon)
	kryui.UnloadImage(icon)
	kryui.SetThemeStyle(kryui.ThemeStyleMaterial)
	kryui.SetCurrentTheme(11, true) // Cobalt dark
	kryui.EnsureUIDefaultFont()
	// The emoji fingerprint font serves only the master-password emoji
	// codepoints; all other glyphs keep coming from the theme font.
	kryui.RegisterUIFixedFontData("gopass-emoji", ".ttf", emojiFontTTF, password.MasterEmojiCodepoints())
	kryui.SetWindowMinSize(560, 620)
	kryui.EnableEventWaiting()

	a := newApp()
	defer func() { a.master.Clear(); a.clipboard.clear() }()
	for !kryui.WindowShouldClose() {
		a.clipboard.tick()
		kryui.BeginDrawing()
		w := kryui.GetScreenWidth()
		kryui.BeginUI(kryui.Key("gopass/main"))
		a.draw(w)
		kryui.EndUI()
		a.handleUIEvents()
		kryui.EndDrawing()
	}
}

func (a *app) draw(width int32) {
	bg, text := kryui.GetThemeBackground(), kryui.GetThemeText()
	scheme := kryui.GetUIMaterialScheme()
	kryui.ClearBackground(bg)
	contentW := width - 64
	if contentW > 720 {
		contentW = 720
	}
	x := (width - contentW) / 2
	kryui.DrawRectangleRounded(kryui.NewRectangle(float32(x), 22, 48, 48), .22, 10, scheme.Primary)
	kryui.DrawUIText("g", x+15, 29, kryui.UIText32, scheme.OnPrimary)
	kryui.DrawUIText("gopass", x+64, 24, kryui.UIText32, text)
	kryui.DrawUIText("Private by design · generated on this device", x+64, 60, kryui.UIText14, scheme.OnSurfaceVariant)
	kryui.DrawRectangleRounded(kryui.NewRectangle(float32(x), 92, float32(contentW), 550), .035, 10, scheme.Surface)
	kryui.DrawRectangleLinesEx(kryui.NewRectangle(float32(x), 92, float32(contentW), 550), 1, scheme.Outline)
	kryui.DrawUIText("PASSWORD DETAILS", x+24, 116, kryui.UIText12, scheme.Primary)

	style := inputStyle()
	y := int32(146)
	a.field("Site", a.site, x+24, y, contentW-48, style)
	y += 76
	a.field("Login", a.login, x+24, y, contentW-48, style)
	y += 76
	kryui.DrawUIText("Master password", x+24, y, kryui.UIText14, text)
	a.master.SetSecure(!a.reveal)
	kryui.TextField(a.master, kryui.FieldProps{Bounds: kryui.NewRectangle(float32(x+24), float32(y+24), float32(contentW-150), 38), Font: kryui.UIText16, Style: style})
	if a.fingerprint != "" {
		fontToken := kryui.PushUIFont("gopass-emoji")
		width := kryui.MeasureText(a.fingerprint, kryui.UIText16)
		kryui.DrawUIText(a.fingerprint, x+contentW-24-width, y, kryui.UIText16, scheme.OnSurfaceVariant)
		kryui.PopUIFont(fontToken)
	}
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+contentW-112), float32(y+24), 88, 38), Label: map[bool]string{true: "Hide", false: "Reveal"}[a.reveal], Style: kryui.UIButtonStyleSecondary, ID: 205})
	y += 76

	kryui.DrawUIText("PASSWORD RULES", x+24, y-10, kryui.UIText12, scheme.Primary)
	y += 14
	kryui.DrawUIText("Length", x+24, y, kryui.UIText14, text)
	kryui.Spinbox(kryui.SpinboxProps{Bounds: kryui.NewRectangle(float32(x+24), float32(y+24), 130, 38), ID: 301, Min: 1, Max: 128, Step: 1, Value: &a.length})
	kryui.DrawUIText("Counter", x+182, y, kryui.UIText14, text)
	kryui.Spinbox(kryui.SpinboxProps{Bounds: kryui.NewRectangle(float32(x+182), float32(y+24), 130, 38), ID: 302, Min: 1, Max: 999999, Step: 1, Value: &a.counter})
	kryui.TextField(a.exclude, kryui.FieldProps{Bounds: kryui.NewRectangle(float32(x+340), float32(y+24), float32(contentW-364), 38), Font: kryui.UIText16, Style: style})
	kryui.DrawUIText("Excluded characters", x+340, y, kryui.UIText14, text)
	y += 76

	kryui.DrawUICheckboxToggle(x+24, y, "Lowercase", &a.lower)
	kryui.DrawUICheckboxToggle(x+174, y, "Uppercase", &a.upper)
	kryui.DrawUICheckboxToggle(x+324, y, "Digits", &a.digits)
	kryui.DrawUICheckboxToggle(x+444, y, "Symbols", &a.symbols)
	y += 48
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+24), float32(y), 150, 42), Label: "Generate", Style: kryui.UIButtonStylePrimary, ID: 401})
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+190), float32(y), 150, 42), Label: "Copy for 20s", Style: kryui.UIButtonStyleSecondary, ID: 402, Disabled: a.generated == ""})
	kryui.Button(kryui.ButtonProps{Bounds: kryui.NewRectangle(float32(x+356), float32(y), 120, 42), Label: "Clear", Style: kryui.UIButtonStyleSecondary, ID: 403})
	y += 56
	kryui.DrawRectangleRounded(kryui.NewRectangle(float32(x+24), float32(y), float32(contentW-48), 72), .08, 10, scheme.SurfaceContainer)
	if a.generated != "" {
		kryui.SelectableText(a.generated, x+42, y+15, kryui.UIText20, text)
	} else {
		kryui.DrawUIText("Your generated password appears here", x+42, y+17, kryui.UIText16, scheme.OnSurfaceVariant)
	}
	if a.message != "" {
		kryui.DrawUIText(a.message, x+42, y+45, kryui.UIText12, scheme.OnSurfaceVariant)
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
					a.clipboard.copy(a.generated, clipboardLifetime)
					a.message = "Copied; clipboard clears in 20 seconds"
				}
			case 403:
				a.master.Clear()
				a.fingerprint = ""
				a.generated = ""
				a.message = "Cleared"
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
	a.generated, a.message = result, "Password generated locally"
}

func inputStyle() kryui.UITextInputStyle {
	s := kryui.GetUIMaterialScheme()
	return kryui.UITextInputStyle{Background: s.SurfaceContainer, Border: s.Outline, FocusBorder: s.Primary, Text: s.OnSurface, Cursor: s.Primary, Radius: 6, PaddingX: 10, PaddingY: 8}
}
