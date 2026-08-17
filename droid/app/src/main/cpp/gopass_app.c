/* C port of the gopass desktop GUI (gui/main.go) for the Android build.
 * The desktop branch keeps the Go layout; this port adds a narrow-screen
 * layout for phones: single column, two-abreast rules, scrollable card. */

#include "gopass_app.h"
#include "gopass_core.h"

#include "kryon.h"
#include "embedded_assets.h"
#include "ui_scroll.h"
#include "ui_scaling.h"
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

#define CLIPBOARD_LIFETIME_MS 20000

typedef struct {
    char buffer[1024];
    int cursor;
    int focused;
    int commit;
    int focus_id;
    int max_codepoints;
    int secure;
} Field;

static void
field_init(Field *f, int focus_id, int capacity_codepoints)
{
    memset(f, 0, sizeof(*f));
    f->focus_id = focus_id;
    f->max_codepoints = capacity_codepoints;
}

static const char *
field_text(const Field *f)
{
    return f->buffer;
}

static void
field_clear(Field *f)
{
    memset(f->buffer, 0, sizeof(f->buffer));
    f->cursor = 0;
}

struct GopassApp {
    Field site, login, master, exclude;
    int length, counter;
    int lower, upper, digits, symbols;
    int reveal;
    char generated[136];
    char message[160];
    /* clipboard lease */
    char clip_value[136];
    int64_t clip_expires_ms;
    int clip_has_value;
    int scroll_offset;
};

static int64_t
now_ms(void)
{
    struct timespec ts;

    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
}

static void
clipboard_copy(GopassApp *a, const char *value)
{
    SetClipboardText(value);
    snprintf(a->clip_value, sizeof(a->clip_value), "%s", value);
    a->clip_expires_ms = now_ms() + CLIPBOARD_LIFETIME_MS;
    a->clip_has_value = 1;
}

static void
clipboard_tick(GopassApp *a)
{
    const char *current;

    if(!a->clip_has_value || now_ms() < a->clip_expires_ms)
        return;
    current = GetClipboardText();
    if(current != NULL && strcmp(current, a->clip_value) == 0)
        SetClipboardText("");
    a->clip_has_value = 0;
    a->clip_value[0] = '\0';
}

static void
clipboard_clear(GopassApp *a)
{
    const char *current;

    if(a->clip_has_value) {
        current = GetClipboardText();
        if(current != NULL && strcmp(current, a->clip_value) == 0)
            SetClipboardText("");
    }
    a->clip_has_value = 0;
    a->clip_value[0] = '\0';
}

static void
generate(GopassApp *a)
{
    GopassOptions options;
    char err[160];

    memset(&options, 0, sizeof(options));
    options.length = a->length;
    options.counter = (uint64_t)a->counter;
    options.lowercase = a->lower;
    options.uppercase = a->upper;
    options.digits = a->digits;
    options.symbols = a->symbols;
    options.exclude = a->exclude.buffer;

    if(gopass_generate(field_text(&a->site), field_text(&a->login),
                       field_text(&a->master), &options,
                       a->generated, sizeof(a->generated),
                       err, sizeof(err)) != 0) {
        a->generated[0] = '\0';
        snprintf(a->message, sizeof(a->message), "%s", err);
        return;
    }
    snprintf(a->message, sizeof(a->message), "%s", "Password generated locally");
}

static UITextInputStyle
input_style(void)
{
    UIMaterialScheme s = GetUIMaterialScheme();
    UITextInputStyle style;

    memset(&style, 0, sizeof(style));
    style.background = s.surface_container;
    style.border = s.outline;
    style.focus_border = s.primary;
    style.text = s.on_surface;
    style.cursor = s.primary;
    style.radius = 6.0f;
    style.padding_x = 10;
    style.padding_y = 8;
    return style;
}

static int
draw_field(Field *f, Rectangle bounds, int font, UITextInputStyle style)
{
    TextFieldProps props;

    memset(&props, 0, sizeof(props));
    props.bounds = bounds;
    props.text = f->buffer;
    props.text_size = sizeof(f->buffer);
    props.cursor_position = &f->cursor;
    props.focused = &f->focused;
    props.max_codepoints = f->max_codepoints;
    props.font = font;
    props.focus_id = f->focus_id;
    props.style = style;
    props.commit_pressed = &f->commit;
    props.secure = f->secure;
    return DrawUITextField(props);
}

static int
button(int x, int y, int w, int h, const char *label, UIButtonStyle style, int disabled)
{
    int hover = 0;

    return DrawUIGenericButton(x, y, w, h, label, style, disabled, &hover);
}

static void
checkbox(int x, int y, const char *label, int *value)
{
    DrawUICheckboxToggle(x, y, label, value);
}

static Rectangle
scaled_rect(float x, float y, float w, float h)
{
    Rectangle r;

    r.x = (float)ScaleUIPx((int)x);
    r.y = (float)ScaleUIPx((int)y);
    r.width = (float)ScaleUIPx((int)w);
    r.height = (float)ScaleUIPx((int)h);
    return r;
}

static void
label_text(const char *text, int x, int y, int font, Color color)
{
    Text(text, ScaleUIPx(x), ScaleUIPx(y), font, color);
}

GopassApp *
gopass_app(void)
{
    static GopassApp app;
    static int initialized;

    if(!initialized) {
        initialized = 1;
        field_init(&app.site, 101, 256);
        field_init(&app.login, 102, 256);
        field_init(&app.master, 103, 1024);
        field_init(&app.exclude, 104, 128);
        app.length = 16;
        app.counter = 1;
        app.lower = app.upper = app.digits = app.symbols = 1;
    }
    return &app;
}

/* ------------------------------------------------------------------ */
/* Wide layout: mirrors gui/main.go                                    */
/* ------------------------------------------------------------------ */

static void
draw_wide(GopassApp *a, int width, int height, int top_reserved)
{
    UIMaterialScheme scheme = GetUIMaterialScheme();
    Color text = GetThemeText();
    UITextInputStyle style = input_style();
    int content_w = width - 64;
    int x, y;

    if(content_w > 720)
        content_w = 720;
    x = (width - content_w) / 2;
    y = 22 + top_reserved;

    DrawRectangleRounded(scaled_rect((float)x, (float)y, 48, 48), 0.22f, 10, scheme.primary);
    label_text("g", x + 15, y + 7, 32, scheme.on_primary);
    label_text("gopass", x + 64, y + 2, 32, text);
    label_text("Private by design - generated on this device",
               x + 64, y + 38, 14, scheme.on_surface_variant);
    y += 70;
    DrawRectangleRounded(scaled_rect((float)x, (float)y, (float)content_w, 550), 0.035f, 10, scheme.surface);
    DrawRectangleLinesEx(scaled_rect((float)x, (float)y, (float)content_w, 550), 1.0f, scheme.outline);
    label_text("PASSWORD DETAILS", x + 24, y + 24, 12, scheme.primary);

    y += 54;
    label_text("Site", x + 24, y, 14, text);
    draw_field(&a->site, scaled_rect((float)x + 24, (float)y + 24, (float)content_w - 48, 38), 16, style);
    y += 76;
    label_text("Login", x + 24, y, 14, text);
    draw_field(&a->login, scaled_rect((float)x + 24, (float)y + 24, (float)content_w - 48, 38), 16, style);
    y += 76;
    label_text("Master password", x + 24, y, 14, text);
    a->master.secure = !a->reveal;
    draw_field(&a->master, scaled_rect((float)x + 24, (float)y + 24, (float)content_w - 150, 38), 16, style);
    if(button(ScaleUIPx(x + content_w - 112), ScaleUIPx(y + 24), ScaleUIPx(88), ScaleUIPx(38),
              a->reveal ? "Hide" : "Reveal", UI_BUTTON_STYLE_SECONDARY, 0))
        a->reveal = !a->reveal;
    y += 76;

    label_text("PASSWORD RULES", x + 24, y - 10, 12, scheme.primary);
    y += 14;
    label_text("Length", x + 24, y, 14, text);
    {
        SpinboxProps props;

        memset(&props, 0, sizeof(props));
        props.bounds = scaled_rect((float)x + 24, (float)y + 24, 130, 38);
        props.id = 301;
        props.min = 1;
        props.max = 128;
        props.step = 1;
        props.value = &a->length;
        Spinbox(props);
    }
    label_text("Counter", x + 182, y, 14, text);
    {
        SpinboxProps props;

        memset(&props, 0, sizeof(props));
        props.bounds = scaled_rect((float)x + 182, (float)y + 24, 130, 38);
        props.id = 302;
        props.min = 1;
        props.max = 999999;
        props.step = 1;
        props.value = &a->counter;
        Spinbox(props);
    }
    draw_field(&a->exclude, scaled_rect((float)x + 340, (float)y + 24, (float)content_w - 364, 38), 16, style);
    label_text("Excluded characters", x + 340, y, 14, text);
    y += 76;

    checkbox(ScaleUIPx(x + 24), ScaleUIPx(y), "Lowercase", &a->lower);
    checkbox(ScaleUIPx(x + 174), ScaleUIPx(y), "Uppercase", &a->upper);
    checkbox(ScaleUIPx(x + 324), ScaleUIPx(y), "Digits", &a->digits);
    checkbox(ScaleUIPx(x + 444), ScaleUIPx(y), "Symbols", &a->symbols);
    y += 48;

    if(button(ScaleUIPx(x + 24), ScaleUIPx(y), ScaleUIPx(150), ScaleUIPx(42),
              "Generate", UI_BUTTON_STYLE_PRIMARY, 0))
        generate(a);
    if(button(ScaleUIPx(x + 190), ScaleUIPx(y), ScaleUIPx(150), ScaleUIPx(42),
              "Copy for 20s", UI_BUTTON_STYLE_SECONDARY, a->generated[0] == '\0')) {
        clipboard_copy(a, a->generated);
        snprintf(a->message, sizeof(a->message), "%s", "Copied; clipboard clears in 20 seconds");
    }
    if(button(ScaleUIPx(x + 356), ScaleUIPx(y), ScaleUIPx(120), ScaleUIPx(42),
              "Clear", UI_BUTTON_STYLE_SECONDARY, 0)) {
        field_clear(&a->master);
        a->generated[0] = '\0';
        snprintf(a->message, sizeof(a->message), "%s", "Cleared");
    }
    y += 56;

    DrawRectangleRounded(scaled_rect((float)x + 24, (float)y, (float)content_w - 48, 72), 0.08f, 10, scheme.surface_container);
    if(a->generated[0] != '\0')
        label_text(a->generated, x + 42, y + 15, 20, text);
    else
        label_text("Your generated password appears here", x + 42, y + 17, 16, scheme.on_surface_variant);
    if(a->message[0] != '\0')
        label_text(a->message, x + 42, y + 45, 12, scheme.on_surface_variant);
}

/* ------------------------------------------------------------------ */
/* Narrow phone layout: single column inside a scrollable card         */
/* ------------------------------------------------------------------ */

static void
draw_narrow(GopassApp *a, int width, int height, int top_reserved, int bottom_reserved)
{
    UIMaterialScheme scheme = GetUIMaterialScheme();
    Color text = GetThemeText();
    UITextInputStyle style = input_style();
    UIScrollArea area;
    UIScrollView view;
    int margin = 12;
    int card_x = margin;
    int card_y = top_reserved + 62;
    int card_w = width - margin * 2;
    int card_h = height - bottom_reserved - margin - card_y;
    int pad = 18;
    int inner_x = card_x + pad;
    int inner_w = card_w - pad * 2;
    int half_w = (inner_w - 12) / 2;
    int content_h;
    int x, y, cx, cy;
    Rectangle card;

    if(card_h < 200)
        card_h = 200;

    /* compact header */
    DrawRectangleRounded(scaled_rect((float)margin, (float)top_reserved + 8, 40, 40), 0.22f, 10, scheme.primary);
    label_text("g", margin + 13, top_reserved + 15, 24, scheme.on_primary);
    label_text("gopass", margin + 52, top_reserved + 6, 24, text);
    label_text("Private by design \xc2\xb7 generated on this device",
               margin + 52, top_reserved + 36, 12, scheme.on_surface_variant);

    /* content height (UI units): sections and rows below */
    content_h = 20 /* section label */
        + 3 * 66  /* site, login, master rows */
        + 20 + 62 /* rules label + spinbox row */
        + 66      /* excluded characters */
        + 2 * 36  /* checkbox rows */
        + 50      /* generate */
        + 50      /* copy/clear row */
        + 78      /* result box */
        + pad;

    card = scaled_rect((float)card_x, (float)card_y, (float)card_w, (float)card_h);
    DrawRectangleRounded(card, 0.035f, 10, scheme.surface);
    DrawRectangleLinesEx(card, 1.0f, scheme.outline);

    memset(&area, 0, sizeof(area));
    area.bounds = card;
    area.content_height = ScaleUIPx(content_h);
    area.content_x = ScaleUIPx(inner_x);
    area.content_width = ScaleUIPx(inner_w);
    area.scroll_offset = &a->scroll_offset;
    view = BeginUIScrollContainer(area);
    cx = view.content_x;
    cy = view.content_y;
    /* positions in UI units relative to content origin */
    x = 0;
    y = 0;

    label_text("PASSWORD DETAILS", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 12, scheme.primary);
    y += 20;

    label_text("Site", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 14, text);
    draw_field(&a->site, (Rectangle){(float)(cx + ScaleUIPx(x)), (float)(cy + ScaleUIPx(y + 22)),
                                     (float)ScaleUIPx(inner_w), (float)ScaleUIPx(38)}, 16, style);
    y += 66;

    label_text("Login", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 14, text);
    draw_field(&a->login, (Rectangle){(float)(cx + ScaleUIPx(x)), (float)(cy + ScaleUIPx(y + 22)),
                                      (float)ScaleUIPx(inner_w), (float)ScaleUIPx(38)}, 16, style);
    y += 66;

    label_text("Master password", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 14, text);
    a->master.secure = !a->reveal;
    {
        int field_w = inner_w - 80;

        draw_field(&a->master, (Rectangle){(float)(cx + ScaleUIPx(x)), (float)(cy + ScaleUIPx(y + 22)),
                                           (float)ScaleUIPx(field_w), (float)ScaleUIPx(38)}, 16, style);
        if(button(cx + ScaleUIPx(x + field_w + 8), cy + ScaleUIPx(y + 22), ScaleUIPx(72), ScaleUIPx(38),
                  a->reveal ? "Hide" : "Reveal", UI_BUTTON_STYLE_SECONDARY, 0))
            a->reveal = !a->reveal;
    }
    y += 66;

    label_text("PASSWORD RULES", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 12, scheme.primary);
    y += 18;
    label_text("Length", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 14, text);
    {
        SpinboxProps props;

        memset(&props, 0, sizeof(props));
        props.bounds = (Rectangle){(float)(cx + ScaleUIPx(x)), (float)(cy + ScaleUIPx(y + 20)),
                                   (float)ScaleUIPx(half_w), (float)ScaleUIPx(38)};
        props.id = 301;
        props.min = 1;
        props.max = 128;
        props.step = 1;
        props.value = &a->length;
        Spinbox(props);
    }
    label_text("Counter", cx + ScaleUIPx(x + half_w + 12), cy + ScaleUIPx(y), 14, text);
    {
        SpinboxProps props;

        memset(&props, 0, sizeof(props));
        props.bounds = (Rectangle){(float)(cx + ScaleUIPx(x + half_w + 12)), (float)(cy + ScaleUIPx(y + 20)),
                                   (float)ScaleUIPx(half_w), (float)ScaleUIPx(38)};
        props.id = 302;
        props.min = 1;
        props.max = 999999;
        props.step = 1;
        props.value = &a->counter;
        Spinbox(props);
    }
    y += 62;

    label_text("Excluded characters", cx + ScaleUIPx(x), cy + ScaleUIPx(y), 14, text);
    draw_field(&a->exclude, (Rectangle){(float)(cx + ScaleUIPx(x)), (float)(cy + ScaleUIPx(y + 22)),
                                        (float)ScaleUIPx(inner_w), (float)ScaleUIPx(38)}, 16, style);
    y += 66;

    checkbox(cx + ScaleUIPx(x), cy + ScaleUIPx(y), "Lowercase", &a->lower);
    checkbox(cx + ScaleUIPx(x + half_w + 12), cy + ScaleUIPx(y), "Uppercase", &a->upper);
    y += 36;
    checkbox(cx + ScaleUIPx(x), cy + ScaleUIPx(y), "Digits", &a->digits);
    checkbox(cx + ScaleUIPx(x + half_w + 12), cy + ScaleUIPx(y), "Symbols", &a->symbols);
    y += 44;

    if(button(cx + ScaleUIPx(x), cy + ScaleUIPx(y), ScaleUIPx(inner_w), ScaleUIPx(42),
              "Generate", UI_BUTTON_STYLE_PRIMARY, 0))
        generate(a);
    y += 50;
    if(button(cx + ScaleUIPx(x), cy + ScaleUIPx(y), ScaleUIPx(half_w), ScaleUIPx(42),
              "Copy for 20s", UI_BUTTON_STYLE_SECONDARY, a->generated[0] == '\0')) {
        clipboard_copy(a, a->generated);
        snprintf(a->message, sizeof(a->message), "%s", "Copied; clipboard clears in 20 seconds");
    }
    if(button(cx + ScaleUIPx(x + half_w + 12), cy + ScaleUIPx(y), ScaleUIPx(half_w), ScaleUIPx(42),
              "Clear", UI_BUTTON_STYLE_SECONDARY, 0)) {
        field_clear(&a->master);
        a->generated[0] = '\0';
        snprintf(a->message, sizeof(a->message), "%s", "Cleared");
    }
    y += 50;

    DrawRectangleRounded((Rectangle){(float)(cx + ScaleUIPx(x)), (float)(cy + ScaleUIPx(y)),
                                     (float)ScaleUIPx(inner_w), (float)ScaleUIPx(72)}, 0.08f, 10,
                         scheme.surface_container);
    if(a->generated[0] != '\0')
        DrawUITextInRect(a->generated,
                         (Rectangle){(float)(cx + ScaleUIPx(x + 14)), (float)(cy + ScaleUIPx(y + 8)),
                                     (float)ScaleUIPx(inner_w - 28), (float)ScaleUIPx(40)},
                         16, text);
    else
        label_text("Your generated password appears here", x + 14, y + 14, 14, scheme.on_surface_variant);
    if(a->message[0] != '\0')
        label_text(a->message, x + 14, y + 50, 11, scheme.on_surface_variant);

    EndUIScrollContainer(area, view);
}

void
gopass_app_draw(GopassApp *a, int surface_w, int surface_h, float dpi,
                int top_reserved, int bottom_reserved)
{
    int ui_w = (int)(surface_w / (dpi > 0.0f ? dpi : 1.0f) + 0.5f);
    int ui_h = (int)(surface_h / (dpi > 0.0f ? dpi : 1.0f) + 0.5f);

    (void)ui_h;
    Background(GetThemeBackground());
    clipboard_tick(a);
    if(ui_w >= 560)
        draw_wide(a, ui_w, ui_h, top_reserved);
    else
        draw_narrow(a, ui_w, ui_h, top_reserved, bottom_reserved);
}

void
gopass_app_shutdown(GopassApp *a)
{
    if(a == NULL)
        return;
    field_clear(&a->master);
    clipboard_clear(a);
    a->generated[0] = '\0';
}
