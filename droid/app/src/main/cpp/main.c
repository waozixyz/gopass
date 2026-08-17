/* Android entry point for gopass. Raylib's Android backend provides
 * android_main and calls main(); this file owns window, theme, fonts, and
 * the frame loop. */

#include "kryon.h"
#include "embedded_assets.h"
#include "ui_dpi.h"
#include "ui_core.h"
#include "ui_scaling.h"

#include "android_bridge.h"
#include "gopass_app.h"

#include <stdio.h>
#include <string.h>

#if ANDROID_BUILD
#include <android/log.h>
#include <unistd.h>
#endif

static const char *const FONT_ASSET_PATH = "vendor/kryon/fonts/noto/NotoSans-Regular.ttf";

/* Kryon's shape drawing rides on a 1x1 white texture so rectangles tint
 * cleanly on the GL ES surface (same setup inbe performs on Android). */
static void
setup_shapes_texture(void)
{
    Image white = GenImageColor(1, 1, WHITE);
    Texture2D texture = LoadTextureFromImage(white);

    UnloadImage(white);
    if(texture.id == 0)
        return;
    SetTextureFilter(texture, TEXTURE_FILTER_POINT);
    SetShapesTexture(texture, (Rectangle){0.0f, 0.0f, 1.0f, 1.0f});
}

static int
setup_ui_font(void)
{
    const EmbeddedAsset *asset = GetEmbeddedAsset(FONT_ASSET_PATH);

    if(asset == NULL || asset->data == NULL || asset->size == 0) {
        TraceLog(LOG_WARNING, "GOPASS: missing embedded font asset");
        return 0;
    }
    if(!RegisterUIFontSource("ui", GetEmbeddedAssetExtension(FONT_ASSET_PATH),
                             asset->data, asset->size, NULL, 0)) {
        TraceLog(LOG_WARNING, "GOPASS: RegisterUIFontSource failed");
        return 0;
    }
    if(!UseUIFont("ui")) {
        TraceLog(LOG_WARNING, "GOPASS: UseUIFont failed");
        return 0;
    }
    return 1;
}

int
main(int argc, char **argv)
{
    GopassApp *app;

    (void)argc;
    (void)argv;

#if ANDROID_BUILD
    __android_log_write(ANDROID_LOG_INFO, "GOPASS_MAIN", "main start");
    android_bridge_init();
    if(chdir("/data/user/0/xyz.waozi.gopass/files") != 0)
        TraceLog(LOG_WARNING, "GOPASS: failed to switch to files directory");
#endif

    InitWindow(0, 0, "gopass");
    if(!IsWindowReady()) {
        TraceLog(LOG_ERROR, "GOPASS: InitWindow failed");
        return 1;
    }
    InitUIDPI();
    SetThemeStyle(THEME_STYLE_MATERIAL);
    SetCurrentTheme(11, 1); /* Cobalt dark, same as the desktop GUI */
    setup_ui_font();
    setup_shapes_texture();
    SetUITextInputPlatformCallback(android_bridge_set_soft_keyboard);
    SetTargetFPS(60);

    app = gopass_app();
    while(!WindowShouldClose()) {
        int width = GetScreenWidth();
        int height = GetScreenHeight();
        Vector2 scale = GetWindowScaleDPI();
        float dpi = scale.x > 0.0f ? scale.x : 1.0f;

        BeginDrawing();
        BeginUIFrame(width, height, dpi);
        gopass_app_draw(app, width, height, dpi,
                        android_bridge_top_reserved(),
                        android_bridge_bottom_reserved());
        EndUIFrame();
        EndDrawing();
    }

    gopass_app_shutdown(app);
    CloseWindow();
    return 0;
}
