/* Minimal JNI bridge for the gopass Android app: safe-area insets, display
 * density, soft-keyboard visibility, and soft-keyboard text input. Modeled
 * on inbe's android_device.c / android_insets.c. */

#include "android_bridge.h"

#if ANDROID_BUILD

#include "kryon.h"
#include "ui_dpi.h"

#include <android/log.h>
#include <android_native_app_glue.h>
#include <jni.h>
#include <pthread.h>

extern struct android_app *GetAndroidApp(void);

#ifndef JNI_VERSION_1_6
#define JNI_VERSION_1_6 0x10060000
#endif

#define LOG_TAG "GOPASS_JNI"

static pthread_mutex_t bridge_mutex = PTHREAD_MUTEX_INITIALIZER;
static int insets_status_bar = 0;
static int insets_nav_bar = 0;
static int insets_cutout_top = 0;
static int insets_cutout_bottom = 0;
static int insets_ready = 0;
static float device_density = 0.0f;

static int
scaled_inset(int java_px, float density)
{
    if(density <= 0.0f)
        density = 1.0f;
    return (int)(java_px / density + 0.5f);
}

void
android_bridge_init(void)
{
    pthread_mutex_lock(&bridge_mutex);
    insets_status_bar = 0;
    insets_nav_bar = 0;
    insets_cutout_top = 0;
    insets_cutout_bottom = 0;
    insets_ready = 0;
    device_density = 0.0f;
    pthread_mutex_unlock(&bridge_mutex);
}

int
android_bridge_top_reserved(void)
{
    int status, cutout, top, ready;
    float density;

    pthread_mutex_lock(&bridge_mutex);
    status = insets_status_bar;
    cutout = insets_cutout_top;
    density = device_density;
    ready = insets_ready;
    pthread_mutex_unlock(&bridge_mutex);

    if(!ready)
        return 28; /* conservative status-bar guess until Java reports */
    top = status > cutout ? status : cutout;
    return scaled_inset(top, density);
}

int
android_bridge_bottom_reserved(void)
{
    int nav, cutout, bottom, ready;
    float density;

    pthread_mutex_lock(&bridge_mutex);
    nav = insets_nav_bar;
    cutout = insets_cutout_bottom;
    density = device_density;
    ready = insets_ready;
    pthread_mutex_unlock(&bridge_mutex);

    if(!ready)
        return 48; /* conservative nav-bar guess until Java reports */
    bottom = nav > cutout ? nav : cutout;
    return scaled_inset(bottom, density);
}

void
android_bridge_set_soft_keyboard(int visible)
{
    struct android_app *app = GetAndroidApp();
    JavaVM *jvm;
    JNIEnv *env = NULL;
    jobject activity;
    jclass activity_class;
    jmethodID method;
    int attached = 0;

    if(app == NULL || app->activity == NULL || app->activity->vm == NULL ||
       app->activity->clazz == NULL)
        return;

    jvm = app->activity->vm;
    activity = app->activity->clazz;
    if((*jvm)->GetEnv(jvm, (void **)&env, JNI_VERSION_1_6) != JNI_OK) {
        if((*jvm)->AttachCurrentThread(jvm, &env, NULL) != JNI_OK || env == NULL)
            return;
        attached = 1;
    }

    activity_class = (*env)->GetObjectClass(env, activity);
    if(activity_class == NULL)
        goto done;

    method = (*env)->GetMethodID(env, activity_class, "setSoftKeyboardVisible", "(Z)V");
    if(method == NULL) {
        __android_log_write(ANDROID_LOG_ERROR, LOG_TAG, "setSoftKeyboardVisible not found");
        goto done;
    }

    (*env)->CallVoidMethod(env, activity, method, visible ? JNI_TRUE : JNI_FALSE);

done:
    if((*env)->ExceptionCheck(env)) {
        (*env)->ExceptionDescribe(env);
        (*env)->ExceptionClear(env);
    }
    if(attached)
        (*jvm)->DetachCurrentThread(jvm);
}

/* ---- Java -> C natives (exported JNI symbols; RegisterNatives from
 * JNI_OnLoad resolves through the boot loader here and does not bind to the
 * activity's class, so we rely on standard dlsym resolution instead) ---- */

JNIEXPORT void JNICALL
Java_xyz_waozi_gopass_MainActivity_nativeSetInsets(JNIEnv *env, jobject thiz,
                                                   jint status_bar, jint nav_bar,
                                                   jint cutout_left, jint cutout_top,
                                                   jint cutout_right, jint cutout_bottom)
{
    (void)env;
    (void)thiz;
    (void)cutout_left;
    (void)cutout_right;

    pthread_mutex_lock(&bridge_mutex);
    insets_status_bar = status_bar;
    insets_nav_bar = nav_bar;
    insets_cutout_top = cutout_top;
    insets_cutout_bottom = cutout_bottom;
    insets_ready = 1;
    pthread_mutex_unlock(&bridge_mutex);
}

JNIEXPORT void JNICALL
Java_xyz_waozi_gopass_MainActivity_nativeSetDeviceDensity(JNIEnv *env, jobject thiz, jfloat density)
{
    (void)env;
    (void)thiz;

    if(density <= 0.0f)
        return;
    pthread_mutex_lock(&bridge_mutex);
    device_density = density;
    pthread_mutex_unlock(&bridge_mutex);
    SetUIDeviceDensity(density);
}

JNIEXPORT void JNICALL
Java_xyz_waozi_gopass_MainActivity_nativeTextInputCommit(JNIEnv *env, jobject thiz, jint codepoint)
{
    (void)env;
    (void)thiz;
    QueueUITextInputCodepoint((int)codepoint);
}

JNIEXPORT void JNICALL
Java_xyz_waozi_gopass_MainActivity_nativeTextInputBackspace(JNIEnv *env, jobject thiz)
{
    (void)env;
    (void)thiz;
    QueueUITextInputBackspace();
}

JNIEXPORT void JNICALL
Java_xyz_waozi_gopass_MainActivity_nativeTextInputEnter(JNIEnv *env, jobject thiz)
{
    (void)env;
    (void)thiz;
    QueueUITextInputEnter();
}

#else /* !ANDROID_BUILD */

void android_bridge_init(void) {}
int android_bridge_top_reserved(void) { return 0; }
int android_bridge_bottom_reserved(void) { return 0; }
void android_bridge_set_soft_keyboard(int visible) { (void)visible; }

#endif
