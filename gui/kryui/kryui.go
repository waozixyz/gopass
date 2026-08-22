package kryui

import (
	"strings"
	"unicode/utf8"

	kryon "github.com/waozixyz/kryon/go/kryon"
)

type (
	Color            = kryon.Color
	Rectangle        = kryon.Rectangle
	Texture2D        = kryon.Texture2D
	ButtonProps      = kryon.ButtonProps
	SpinboxProps     = kryon.SpinboxProps
	BottomNavItem    = kryon.BottomNavItem
	ThemeStyle       = kryon.ThemeStyle
	ThemeSource      = kryon.ThemeSource
	ThemeMode        = kryon.ThemeMode
	UITextInputStyle = kryon.TextInputStyle
	UIKey            int32
)

const (
	FlagWindowResizable = kryon.FlagWindowResizable
	ThemeStyleMaterial  = kryon.ThemeStyleMaterial
	ThemeSourceSystem   = kryon.ThemeSourceSystem
	ThemeModeSystem     = kryon.ThemeModeSystem

	UIButtonStylePrimary   = kryon.ButtonStylePrimary
	UIButtonStyleSecondary = kryon.ButtonStyleSecondary

	UIText12 = kryon.Text12
	UIText14 = kryon.Text14
	UIText16 = kryon.Text16
	UIText20 = kryon.Text20

	EventClick eventKind = iota + 1
	EventTextChanged
)

type eventKind int

type UIEvent struct {
	Kind eventKind
	Key  UIKey
}

type FieldProps struct {
	Bounds Rectangle
	Font   int32
	Style  UITextInputStyle
}

type TextFieldState struct {
	buf     []byte
	cursor  int32
	focused bool
	id      UIKey
	secure  bool
}

type BottomNavProps struct {
	ViewWidth  int32
	ViewHeight int32
	Items      []BottomNavItem
}

type BottomNavResult struct {
	ClickedRoute int32
}

type MaterialScheme struct {
	Primary          Color
	Surface          Color
	SurfaceContainer Color
	OnSurface        Color
	OnSurfaceVariant Color
	Outline          Color
}

var events []UIEvent

func NewTextField(id UIKey, maxCodepoints int32) *TextFieldState {
	if maxCodepoints < 1 {
		maxCodepoints = 1
	}
	return &TextFieldState{
		buf: make([]byte, int(maxCodepoints)*4+1),
		id:  id,
	}
}

func NewPasswordField(id UIKey, maxCodepoints int32) *TextFieldState {
	field := NewTextField(id, maxCodepoints)
	field.secure = true
	return field
}

func (f *TextFieldState) Text() string {
	if f == nil {
		return ""
	}
	return string(f.buf[:zeroIndex(f.buf)])
}

func (f *TextFieldState) SetText(text string) {
	if f == nil {
		return
	}
	clear(f.buf)
	copy(f.buf, clampText(text, len(f.buf)-1))
	f.cursor = int32(len(f.Text()))
}

func (f *TextFieldState) Clear() {
	f.SetText("")
}

func (f *TextFieldState) SetSecure(secure bool) {
	if f != nil {
		f.secure = secure
	}
}

func SetConfigFlags(flags uint) { kryon.SetConfigFlags(flags) }
func InitWindow(width, height int32, title string) {
	kryon.InitWindow(width, height, title)
}
func WindowShouldClose() bool                   { return kryon.WindowShouldClose() }
func BeginDrawing()                             { kryon.BeginDrawing() }
func EndDrawing()                               { kryon.EndDrawing() }
func CloseWindow()                              { kryon.CloseWindow() }
func GetScreenWidth() int32                     { return kryon.GetScreenWidth() }
func GetScreenHeight() int32                    { return kryon.GetScreenHeight() }
func ClearBackground(c Color)                   { kryon.ClearBackground(c) }
func GetThemeBackground() Color                 { return kryon.GetThemeBackground() }
func GetThemeText() Color                       { return kryon.GetThemeText() }
func SetThemeStyle(style ThemeStyle)            { kryon.SetThemeStyle(style) }
func SetThemeSource(source ThemeSource)         { kryon.SetThemeSource(source) }
func SetThemeMode(mode ThemeMode)               { kryon.SetThemeMode(mode) }
func NewRectangle(x, y, w, h float32) Rectangle { return kryon.NewRectangle(x, y, w, h) }
func Key(text string) kryon.KeyID               { return kryon.Key(text) }

func SetCurrentTheme(themeID int32, darkMode bool) {
	if darkMode {
		kryon.SetCurrentTheme(themeID, 1)
		return
	}
	kryon.SetCurrentTheme(themeID, 0)
}

func BeginUI(kryon.KeyID) {
	events = events[:0]
}

func EndUI() {}

func NextUIEvent() (UIEvent, bool) {
	if len(events) == 0 {
		return UIEvent{}, false
	}
	event := events[0]
	events = events[1:]
	return event, true
}

func Button(props ButtonProps) bool {
	if props.Font == 0 {
		props.Font = UIText16
	}
	pressed := kryon.Button(props)
	if pressed {
		events = append(events, UIEvent{Kind: EventClick, Key: UIKey(props.ID)})
	}
	return pressed
}

func TextField(field *TextFieldState, props FieldProps) bool {
	if field == nil {
		return false
	}
	before := field.Text()
	kryon.TextField(kryon.TextFieldProps{
		Bounds:         props.Bounds,
		Text:           field.buf,
		CursorPosition: &field.cursor,
		Focused:        &field.focused,
		MaxCodepoints:  int32(capRunes(field.buf)),
		Font:           props.Font,
		FocusID:        int32(field.id),
		Style:          props.Style,
		Secure:         field.secure,
	})
	changed := field.Text() != before
	if changed {
		events = append(events, UIEvent{Kind: EventTextChanged, Key: field.id})
	}
	return changed
}

func BottomNav(props BottomNavProps) BottomNavResult {
	const (
		height = int32(56)
		margin = int32(16)
	)
	count := int32(len(props.Items))
	if count == 0 {
		return BottomNavResult{ClickedRoute: -1}
	}
	width := props.ViewWidth - margin*2
	if width < count {
		width = count
	}
	itemW := width / count
	y := props.ViewHeight - height - margin
	for i, item := range props.Items {
		style := kryon.ButtonStyleSecondary
		if item.Active {
			style = kryon.ButtonStylePrimary
		}
		if Button(ButtonProps{
			Bounds:   NewRectangle(float32(margin+int32(i)*itemW), float32(y), float32(itemW-8), float32(height)),
			Label:    item.Label,
			Style:    style,
			Font:     UIText14,
			ID:       9000 + item.Route,
			Disabled: item.Disabled,
		}) {
			return BottomNavResult{ClickedRoute: item.Route}
		}
	}
	return BottomNavResult{ClickedRoute: -1}
}

func Spinbox(props SpinboxProps) bool {
	return kryon.Spinbox(props)
}

func DrawUICheckboxToggle(x, y int32, label string, value *bool) bool {
	checked := int32(0)
	if value != nil && *value {
		checked = 1
	}
	changed := kryon.Checkbox(int32(kryon.Key(label))&0x7fffffff, x, y, label, &checked)
	if changed && value != nil {
		*value = checked != 0
	}
	return changed
}

func DrawUIText(text string, x, y, fontSize int32, color Color) {
	kryon.Text(text, x, y, fontSize, color)
}

func DrawRectangleRounded(rect Rectangle, _ float32, _ int32, color Color) {
	kryon.DrawRectangleRec(rect, color)
}

func DrawRectangleLinesEx(rect Rectangle, thick int32, color Color) {
	kryon.DrawRectangleLinesEx(rect, thick, color)
}

func SelectableText(value string, x, y, fontSize int32, color Color) {
	kryon.SelectableText(value, x, y, fontSize, color)
}

func MeasureText(text string, fontSize int32) int32 {
	return int32(utf8.RuneCountInString(text)) * fontSize * 11 / 20
}

func GetUIMaterialScheme() MaterialScheme {
	bg := kryon.GetThemeBackground()
	text := kryon.GetThemeText()
	surface := kryon.GetThemeSurface()
	button := kryon.GetThemeButton()
	return MaterialScheme{
		Primary:          button,
		Surface:          surface,
		SurfaceContainer: mix(surface, bg, 0.25),
		OnSurface:        text,
		OnSurfaceVariant: mix(text, bg, 0.3),
		Outline:          mix(text, bg, 0.45),
	}
}

func GetClipboardText() string                                    { return kryon.ClipboardText() }
func SetClipboardText(text string)                                { kryon.SetClipboardText(text) }
func ShowToast(message string)                                    { kryon.ShowToast(message) }
func EnsureUIDefaultFont()                                        {}
func EnableEventWaiting()                                         {}
func SetWindowMinSize(_, _ int32)                                 {}
func LoadImageFromMemory(string, []byte) Texture2D                { return Texture2D{} }
func SetWindowIcon(Texture2D)                                     {}
func UnloadImage(Texture2D)                                       {}
func RegisterUIFixedFontData(string, string, []byte, []rune) bool { return true }
func PushUIFont(string) int                                       { return 0 }
func PopUIFont(int)                                               {}

type UpdateFlowState int

const (
	UpdateFlowAvailable UpdateFlowState = iota + 1
	UpdateFlowDownloading
	UpdateFlowReady
	UpdateFlowFailed
)

type UpdateFlow struct{}

func StartUpdateFlow(string, string, string) *UpdateFlow { return nil }
func (*UpdateFlow) Poll()                                {}
func (*UpdateFlow) State() UpdateFlowState               { return 0 }
func (*UpdateFlow) HasArtifact() bool                    { return false }
func (*UpdateFlow) NewVersion() string                   { return "" }
func (*UpdateFlow) ReleaseURL() string                   { return "" }
func (*UpdateFlow) Download()                            {}
func (*UpdateFlow) Progress() float64                    { return 0 }
func (*UpdateFlow) Apply() bool                          { return false }
func (*UpdateFlow) ExecPending()                         {}
func (*UpdateFlow) Error() string                        { return "" }

func zeroIndex(buf []byte) int {
	for i, b := range buf {
		if b == 0 {
			return i
		}
	}
	return len(buf)
}

func capRunes(buf []byte) int {
	if len(buf) <= 1 {
		return 0
	}
	return (len(buf) - 1) / 4
}

func clampText(text string, maxBytes int) string {
	if len(text) <= maxBytes {
		return text
	}
	var b strings.Builder
	for _, r := range text {
		size := utf8.RuneLen(r)
		if size < 0 || b.Len()+size > maxBytes {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func mix(a, b Color, t float32) Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return Color{
		R: uint8(float32(a.R)*(1-t) + float32(b.R)*t),
		G: uint8(float32(a.G)*(1-t) + float32(b.G)*t),
		B: uint8(float32(a.B)*(1-t) + float32(b.B)*t),
		A: uint8(float32(a.A)*(1-t) + float32(b.A)*t),
	}
}
