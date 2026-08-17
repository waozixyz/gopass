#ifndef GOPASS_APP_H
#define GOPASS_APP_H

typedef struct GopassApp GopassApp;

GopassApp *gopass_app(void);

/* Draws one frame. Surface sizes are physical pixels; dpi is the display
 * scale. top/bottom_reserved are safe-area insets in UI units (dp). */
void gopass_app_draw(GopassApp *app, int surface_w, int surface_h, float dpi,
                     int top_reserved, int bottom_reserved);

/* Clears secrets held in memory. */
void gopass_app_shutdown(GopassApp *app);

#endif
