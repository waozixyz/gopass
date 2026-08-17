package xyz.waozi.pass;

import android.app.NativeActivity;
import android.graphics.Insets;
import android.os.Build;
import android.os.Bundle;
import android.view.DisplayCutout;
import android.view.KeyEvent;
import android.view.View;
import android.view.ViewTreeObserver;
import android.view.WindowInsets;
import android.view.inputmethod.InputMethodManager;
import android.content.Context;

/**
 * NativeActivity glue for the kryon UI: routes soft-keyboard input and
 * window insets to the native side. Modeled on inbe's MainActivity.
 */
public class MainActivity extends NativeActivity {
    static {
        // Associate libmain with this class's loader so ART can resolve the
        // native methods below (NativeActivity loads it via the boot loader).
        System.loadLibrary("main");
    }

    private int lastDeleteRepeatCount = -1;

    private native void nativeSetInsets(int status, int nav,
        int cutoutLeft, int cutoutTop, int cutoutRight, int cutoutBottom);
    private native void nativeSetDeviceDensity(float density);
    private native void nativeTextInputCommit(int codepoint);
    private native void nativeTextInputBackspace();
    private native void nativeTextInputEnter();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setupInsetsListener();
    }

    public void setSoftKeyboardVisible(final boolean visible) {
        runOnUiThread(new Runnable() {
            @Override
            public void run() {
                InputMethodManager imm =
                    (InputMethodManager)getSystemService(Context.INPUT_METHOD_SERVICE);
                View view = getWindow() != null ? getWindow().getDecorView() : null;
                if (imm == null || view == null) return;

                if (visible) {
                    view.requestFocus();
                    imm.showSoftInput(view, InputMethodManager.SHOW_FORCED);
                } else {
                    imm.hideSoftInputFromWindow(view.getWindowToken(), 0);
                }
            }
        });
    }

    @Override
    public boolean dispatchKeyEvent(KeyEvent event) {
        if (event != null && event.getAction() == KeyEvent.ACTION_DOWN) {
            int keyCode = event.getKeyCode();
            if (keyCode == KeyEvent.KEYCODE_DEL) {
                int repeatCount = event.getRepeatCount();
                int deleteCount = lastDeleteRepeatCount < 0
                    ? 1
                    : Math.max(1, repeatCount - lastDeleteRepeatCount);
                lastDeleteRepeatCount = repeatCount;
                for (int i = 0; i < deleteCount; i++) {
                    nativeTextInputBackspace();
                }
            } else if (keyCode == KeyEvent.KEYCODE_ENTER ||
                       keyCode == KeyEvent.KEYCODE_NUMPAD_ENTER) {
                lastDeleteRepeatCount = -1;
                nativeTextInputEnter();
            } else {
                lastDeleteRepeatCount = -1;
                int unicode = event.getUnicodeChar();
                if (unicode >= 32) {
                    nativeTextInputCommit(unicode);
                }
            }
        } else if (event != null && event.getAction() == KeyEvent.ACTION_UP) {
            if (event.getKeyCode() == KeyEvent.KEYCODE_DEL) {
                lastDeleteRepeatCount = -1;
            }
        } else if (event != null && event.getAction() == KeyEvent.ACTION_MULTIPLE &&
                   event.getCharacters() != null) {
            lastDeleteRepeatCount = -1;
            String chars = event.getCharacters();
            for (int i = 0; i < chars.length();) {
                int codepoint = chars.codePointAt(i);
                if (codepoint >= 32) {
                    nativeTextInputCommit(codepoint);
                }
                i += Character.charCount(codepoint);
            }
        }
        return super.dispatchKeyEvent(event);
    }

    private void setupInsetsListener() {
        final View decorView = getWindow().getDecorView();

        nativeSetDeviceDensity(getResources().getDisplayMetrics().density);

        decorView.setOnApplyWindowInsetsListener(new View.OnApplyWindowInsetsListener() {
            @Override
            public WindowInsets onApplyWindowInsets(View v, WindowInsets insets) {
                updateInsets(insets);
                return insets;
            }
        });

        // Startup safety net: catch the initial layout pass.
        decorView.getViewTreeObserver().addOnGlobalLayoutListener(
                new ViewTreeObserver.OnGlobalLayoutListener() {
            @Override
            public void onGlobalLayout() {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                    WindowInsets insets = decorView.getRootWindowInsets();
                    if (insets != null) {
                        updateInsets(insets);
                    }
                }
                decorView.getViewTreeObserver().removeOnGlobalLayoutListener(this);
            }
        });

        decorView.post(new Runnable() {
            @Override
            public void run() {
                decorView.requestApplyInsets();
            }
        });
    }

    private void updateInsets(WindowInsets insets) {
        if (insets == null) return;

        nativeSetDeviceDensity(getResources().getDisplayMetrics().density);

        int statusBar = 0;
        int navBar = 0;
        int cLeft = 0, cTop = 0, cRight = 0, cBottom = 0;

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            Insets systemBars = insets.getInsetsIgnoringVisibility(
                    WindowInsets.Type.systemBars());
            statusBar = systemBars.top;
            navBar = systemBars.bottom;
        } else {
            statusBar = insets.getSystemWindowInsetTop();
            navBar = insets.getSystemWindowInsetBottom();
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            DisplayCutout cutout = insets.getDisplayCutout();
            if (cutout != null) {
                cLeft = cutout.getSafeInsetLeft();
                cTop = cutout.getSafeInsetTop();
                cRight = cutout.getSafeInsetRight();
                cBottom = cutout.getSafeInsetBottom();
            }
        }

        nativeSetInsets(statusBar, navBar, cLeft, cTop, cRight, cBottom);
    }
}
