cmake_minimum_required(VERSION 3.22)
project(Gopass C)

set(REPO_ROOT "${CMAKE_CURRENT_LIST_DIR}/../../../../..")
set(KRYON_DIR "${REPO_ROOT}/vendor/kryon")
set(RAYLIB_DIR "${KRYON_DIR}/vendor/raylib/src")
set(GOPASS_NATIVE_DIR "${REPO_ROOT}/native")
set(NATIVE_APP_GLUE_DIR "${ANDROID_NDK}/sources/android/native_app_glue")

file(GLOB_RECURSE KRYON_SOURCES CONFIGURE_DEPENDS "${KRYON_DIR}/src/*.c")
# The src/backend/ software renderer implements raylib-named functions for
# no-raylib hosts; Android links real raylib, so compiling both is a
# duplicate-symbol link error.
list(FILTER KRYON_SOURCES EXCLUDE REGEX "${KRYON_DIR}/src/backend/.*")
# gopass uses no 2D physics; mirror the desktop Makefile's physics filter.
list(REMOVE_ITEM KRYON_SOURCES
    "${KRYON_DIR}/src/scene/physics_world.c"
    "${KRYON_DIR}/src/scene/node_body2d.c"
    "${KRYON_DIR}/src/scene/node_area2d.c"
    "${KRYON_DIR}/src/scene/node_collision_shape2d.c")
# Ksync account crypto requires liboqs and gopass never syncs; runtime asset
# downloads need an Android download backend gopass does not provide.
list(FILTER KRYON_SOURCES EXCLUDE REGEX "${KRYON_DIR}/src/ksync/.*")
list(FILTER KRYON_SOURCES EXCLUDE REGEX "${KRYON_DIR}/src/runtime_assets/.*")
list(REMOVE_ITEM KRYON_SOURCES "${KRYON_DIR}/src/file_dialog/file_dialog.c")

set(GOPASS_FONT_ASSET "vendor/kryon/fonts/noto/NotoSans-Regular.ttf")
set(GOPASS_EMBEDDED_ASSETS_C "${CMAKE_BINARY_DIR}/gopass_embedded_assets.c")
add_custom_command(
    OUTPUT "${GOPASS_EMBEDDED_ASSETS_C}"
    COMMAND sh "${KRYON_DIR}/scripts/embed-assets.sh" "${GOPASS_EMBEDDED_ASSETS_C}" "${GOPASS_FONT_ASSET}"
    DEPENDS "${KRYON_DIR}/scripts/embed-assets.sh" "${KRYON_DIR}/fonts/noto/NotoSans-Regular.ttf"
    WORKING_DIRECTORY "${REPO_ROOT}"
    VERBATIM
)

add_library(raylib STATIC
    "${RAYLIB_DIR}/rcore.c"
    "${RAYLIB_DIR}/rshapes.c"
    "${RAYLIB_DIR}/rtextures.c"
    "${RAYLIB_DIR}/rtext.c"
    "${RAYLIB_DIR}/raudio.c"
)

target_compile_definitions(raylib PRIVATE
    PLATFORM_ANDROID
    GRAPHICS_API_OPENGL_ES2
    SUPPORT_SCREEN_CAPTURE=0
    SUPPORT_COMPRESSION_API=0
    SUPPORT_AUTOMATION_EVENTS=0
    SUPPORT_CLIPBOARD_IMAGE=0
    SUPPORT_FILEFORMAT_BMP=0
    SUPPORT_FILEFORMAT_GIF=0
    SUPPORT_FILEFORMAT_QOI=0
    SUPPORT_FILEFORMAT_DDS=0
    SUPPORT_FILEFORMAT_TTF=1
    SUPPORT_FILEFORMAT_JPG=1
    SUPPORT_FILEFORMAT_PNG=1
    SUPPORT_FILEFORMAT_OGG=1
    SUPPORT_MODULE_RAUDIO=1
)

target_compile_options(raylib PRIVATE
    -Os
    -ffunction-sections
    -fdata-sections
)

target_include_directories(raylib PUBLIC
    "${RAYLIB_DIR}"
    "${NATIVE_APP_GLUE_DIR}"
)

add_library(main SHARED
    main.c
    gopass_app.c
    android_bridge.c
    "${GOPASS_NATIVE_DIR}/gopass_core.c"
    "${GOPASS_EMBEDDED_ASSETS_C}"
    ${KRYON_SOURCES}
    "${NATIVE_APP_GLUE_DIR}/android_native_app_glue.c"
)

target_include_directories(main PRIVATE
    "${KRYON_DIR}/include"
    "${KRYON_DIR}/src"
    "${GOPASS_NATIVE_DIR}"
    "${NATIVE_APP_GLUE_DIR}"
)

target_compile_definitions(main PRIVATE
    PLATFORM_ANDROID
    ANDROID_BUILD=1
    _DEFAULT_SOURCE
    _GNU_SOURCE
    _FILE_OFFSET_BITS=64
    UI_EMBEDDED_ONLY=1
    KRYON_WITH_PHYSICS=0
)

target_compile_options(main PRIVATE
    -Os
    -ffunction-sections
    -fdata-sections
    -fvisibility=hidden
)

target_link_options(main PRIVATE
    -Wl,--gc-sections
    -Wl,--strip-all
    -Wl,--wrap=fopen
    -Wl,-z,max-page-size=16384
    -Wl,-z,common-page-size=16384
)

target_link_libraries(main
    raylib
    android
    log
    OpenSLES
    EGL
    GLESv2
    m
)
