#ifndef GOPASS_ANDROID_BRIDGE_H
#define GOPASS_ANDROID_BRIDGE_H

#if defined(__cplusplus)
extern "C" {
#endif

/* Call once before InitWindow. */
void android_bridge_init(void);

/* Safe-area insets converted to UI units (java px / density). Returns 0
 * until Java has reported real values; callers should treat 0 as "unknown"
 * and keep a conservative margin. */
int android_bridge_top_reserved(void);
int android_bridge_bottom_reserved(void);

/* Shows or hides the soft keyboard through the activity. */
void android_bridge_set_soft_keyboard(int visible);

#if defined(__cplusplus)
}
#endif

#endif
