/**
 * mpvjni.cpp — JNI bridge entre MpvLib.kt y libmpv.so
 * Basado en mpv-android/app/src/main/cpp/mpvjni.cpp (Apache 2.0)
 */

#include <jni.h>
#include <android/log.h>
#include <mpv/client.h>
#include <mpv/render_gl.h>
#include <string>
#include <atomic>

// av_jni_set_java_vm registra el JVM en libavcodec,
// que libmpv usa internamente para su Android VO context (vo/gpu/android).
// Declaración manual para no depender de los headers de FFmpeg.
extern "C" int av_jni_set_java_vm(void *vm, void *log_ctx);

#define TAG "mpvjni"
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, TAG, __VA_ARGS__)

static mpv_handle *mpv = nullptr;
static jobject surfaceRef = nullptr;   // Global ref to Java Surface (lo usa libmpv internamente)
static JavaVM *jvm = nullptr;
static jobject mpvLibObject = nullptr;

// Clases y métodos JNI cacheados
static jclass mpvLibClass = nullptr;
static jmethodID eventPropertyLongMethod = nullptr;
static jmethodID eventPropertyBoolMethod = nullptr;
static jmethodID eventPropertyDoubleMethod = nullptr;
static jmethodID eventPropertyStringMethod = nullptr;
static jmethodID eventPropertyMethod = nullptr;
static jmethodID eventMethod = nullptr;
static jmethodID logMessageMethod = nullptr;
static jmethodID wakeupMethod = nullptr;

// El wakeup callback NO puede llamar ninguna API de mpv (se ejecuta con locks internos).
// Solo notifica a Kotlin para que procese eventos en su propio hilo.
static void mpv_wakeup_callback(void *ctx) {
    if (!mpvLibClass || !wakeupMethod) return;
    JNIEnv *env;
    int attached = 0;
    jint res = jvm->GetEnv((void **)&env, JNI_VERSION_1_6);
    if (res == JNI_EDETACHED) {
        jvm->AttachCurrentThread(&env, nullptr);
        attached = 1;
    } else if (res != JNI_OK) {
        return;
    }
    env->CallStaticVoidMethod(mpvLibClass, wakeupMethod);
    if (attached) jvm->DetachCurrentThread();
}

// Procesar un evento mpv pendiente. Retorna el eventId (0 = ninguno).
// Llamado desde Kotlin en su hilo de eventos dedicado.
static int processOneMpvEvent(JNIEnv *env) {
    if (!mpv) return 0;
    mpv_event *event = mpv_wait_event(mpv, 0);
    if (event->event_id == MPV_EVENT_NONE) return 0;

    if (event->event_id == MPV_EVENT_LOG_MESSAGE) {
        auto *msg = (mpv_event_log_message *)event->data;
        if (!msg->prefix || !msg->text) return 0;
        jstring prefix = env->NewStringUTF(msg->prefix);
        jstring text = env->NewStringUTF(msg->text);
        if (prefix && text)
            env->CallStaticVoidMethod(mpvLibClass, logMessageMethod, prefix,
                                      (jint)msg->log_level, text);
        if (prefix) env->DeleteLocalRef(prefix);
        if (text) env->DeleteLocalRef(text);
    } else if (event->event_id == MPV_EVENT_PROPERTY_CHANGE) {
        auto *prop = (mpv_event_property *)event->data;
        jstring jname = env->NewStringUTF(prop->name);
        if (jname) {
            if (prop->data && prop->format != MPV_FORMAT_NONE) {
                auto *node = (mpv_node *)prop->data;
                switch (node->format) {
                    case MPV_FORMAT_INT64:
                        env->CallStaticVoidMethod(mpvLibClass, eventPropertyLongMethod, jname,
                                                  (jlong)node->u.int64);
                        break;
                    case MPV_FORMAT_DOUBLE:
                        env->CallStaticVoidMethod(mpvLibClass, eventPropertyDoubleMethod, jname,
                                                  (jdouble)node->u.double_);
                        break;
                    case MPV_FORMAT_FLAG:
                        env->CallStaticVoidMethod(mpvLibClass, eventPropertyBoolMethod, jname,
                                                  (jboolean)(node->u.flag != 0));
                        break;
                    case MPV_FORMAT_STRING:
                        if (node->u.string) {
                            jstring jval = env->NewStringUTF(node->u.string);
                            env->CallStaticVoidMethod(mpvLibClass, eventPropertyStringMethod,
                                                      jname, jval);
                            env->DeleteLocalRef(jval);
                        }
                        break;
                    default:
                        env->CallStaticVoidMethod(mpvLibClass, eventPropertyMethod, jname);
                        break;
                }
            } else {
                env->CallStaticVoidMethod(mpvLibClass, eventPropertyMethod, jname);
            }
            env->DeleteLocalRef(jname);
        }
    } else {
        if (mpvLibClass && eventMethod)
            env->CallStaticVoidMethod(mpvLibClass, eventMethod, (jint)event->event_id);
    }
    return (int)event->event_id;
}

extern "C" {

JNIEXPORT jint JNICALL JNI_OnLoad(JavaVM *vm, void *reserved) {
    jvm = vm;
    // Registrar JVM en libavcodec para que vo/gpu/android pueda inicializar el EGL context
    av_jni_set_java_vm(vm, nullptr);
    return JNI_VERSION_1_6;
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_create(JNIEnv *env, jobject obj, jobject context) {
    mpv = mpv_create();
    if (!mpv) {
        LOGE("mpv_create failed");
        return;
    }

    // Cache JNI references
    mpvLibClass = (jclass)env->NewGlobalRef(env->FindClass("com/p2pollo/mpv/MpvLib"));
    eventPropertyLongMethod = env->GetStaticMethodID(mpvLibClass, "eventProperty", "(Ljava/lang/String;J)V");
    eventPropertyBoolMethod = env->GetStaticMethodID(mpvLibClass, "eventProperty", "(Ljava/lang/String;Z)V");
    eventPropertyDoubleMethod = env->GetStaticMethodID(mpvLibClass, "eventProperty", "(Ljava/lang/String;D)V");
    eventPropertyStringMethod = env->GetStaticMethodID(mpvLibClass, "eventProperty", "(Ljava/lang/String;Ljava/lang/String;)V");
    eventPropertyMethod = env->GetStaticMethodID(mpvLibClass, "eventProperty", "(Ljava/lang/String;)V");
    eventMethod = env->GetStaticMethodID(mpvLibClass, "event", "(I)V");
    logMessageMethod = env->GetStaticMethodID(mpvLibClass, "logMessage", "(Ljava/lang/String;ILjava/lang/String;)V");
    wakeupMethod = env->GetStaticMethodID(mpvLibClass, "wakeup", "()V");

    // Wakeup callback: solo notifica a Kotlin, NO llama API de mpv
    mpv_set_wakeup_callback(mpv, mpv_wakeup_callback, nullptr);
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_init(JNIEnv *env, jobject obj) {
    if (!mpv) return;
    LOGI("init: wid actual = 0x%llx", (unsigned long long)(uintptr_t)surfaceRef);

    mpv_set_option_string(mpv, "vo", "gpu");
    mpv_set_option_string(mpv, "ao", "audiotrack");
    mpv_set_option_string(mpv, "gpu-context", "android");
    mpv_set_option_string(mpv, "opengl-es", "yes");
    // mediacodec (zero-copy): mpv provee la Surface directamente al decoder HEVC.
    // mediacodec-copy falla en Amlogic con HEVC: FFmpeg no puede crear la SurfaceTexture
    // interna → "Both surface and native_window are NULL" → cae a software decode.
    // Con SurfaceView + file://, mpv ya tiene la surface lista antes de decode.
    mpv_set_option_string(mpv, "hwdec", "mediacodec");
    // Reducir trabajo del Mali: bilinear + sin post-procesado
    mpv_set_option_string(mpv, "scale", "bilinear");
    mpv_set_option_string(mpv, "dscale", "bilinear");
    mpv_set_option_string(mpv, "cscale", "bilinear");
    mpv_set_option_string(mpv, "correct-downscaling", "no");
    mpv_set_option_string(mpv, "interpolation", "no");
    // video-sync=audio: sincroniza frames al clock de audio.
    // desync causó audio underruns continuos en Flowbox F1 → A/V desync a los ~10s.
    mpv_set_option_string(mpv, "video-sync", "audio");
    // profile=fast: recomendado por mpv en logs de diagnóstico. Desactiva deband,
    // sigmoid-upscaling y otros efectos caros. Se aplica ANTES de las opciones manuales
    // para que bilinear/correct-downscaling/etc. tengan precedencia sobre el perfil.
    mpv_set_option_string(mpv, "profile", "fast");
    mpv_set_option_string(mpv, "sub-ass", "yes");
    mpv_set_option_string(mpv, "sub-visibility", "yes");
    // sub-font-provider=none: evita que libass intente usar fontconfig (no existe en Android).
    // Sin esto: "can't find selected font provider" y el font matching falla.
    mpv_set_option_string(mpv, "sub-font-provider", "none");
    // sub-font=Roboto: SRT subtitles default to "sans-serif" family name. Sin un font provider,
    // libass no puede resolver nombres genéricos como "sans-serif". Al especificar "Roboto"
    // explícitamente, libass busca esa familia en sub-fonts-dir y encuentra Roboto-Regular.ttf
    // (cuyo nombre interno ES "Roboto"). setupMpvFonts() ya copió este font al dir privado.
    mpv_set_option_string(mpv, "sub-font", "Roboto");
    mpv_set_option_string(mpv, "keep-open", "yes");
    // Cache del demuxer ajustada para 2GB RAM (Flowbox F1):
    // 20M adelante alcanza para absorber variaciones del torrent sin presionar memoria.
    // 5M atrás mínimo para seek reciente. readahead en segundos como tope secundario.
    mpv_set_option_string(mpv, "demuxer-max-bytes", "20M");
    mpv_set_option_string(mpv, "demuxer-max-back-bytes", "5M");
    mpv_set_option_string(mpv, "demuxer-readahead-secs", "15");
    mpv_request_log_messages(mpv, "warn");

    LOGI("init: llamando mpv_initialize...");
    int err = mpv_initialize(mpv);
    LOGI("init: mpv_initialize retornó %d (%s)", err, mpv_error_string(err));
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_destroy(JNIEnv *env, jobject obj) {
    if (!mpv) return;
    mpv_terminate_destroy(mpv);
    mpv = nullptr;
    if (mpvLibClass) {
        env->DeleteGlobalRef(mpvLibClass);
        mpvLibClass = nullptr;
    }
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_attachSurface(JNIEnv *env, jobject obj, jobject surface) {
    // libmpv llama ANativeWindow_fromSurface() internamente sobre el wid.
    // Hay que pasarle el jobject Surface original (como global ref), NO un ANativeWindow*.
    if (surfaceRef) {
        env->DeleteGlobalRef(surfaceRef);
        surfaceRef = nullptr;
    }
    surfaceRef = env->NewGlobalRef(surface);
    if (!mpv) return;
    int64_t wid = (int64_t)(uintptr_t)surfaceRef;
    mpv_set_option(mpv, "wid", MPV_FORMAT_INT64, &wid);
    LOGI("attachSurface: wid=0x%llx", (unsigned long long)wid);
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_detachSurface(JNIEnv *env, jobject obj) {
    if (mpv) {
        int64_t wid = 0;
        mpv_set_option(mpv, "wid", MPV_FORMAT_INT64, &wid);
    }
    if (surfaceRef) {
        env->DeleteGlobalRef(surfaceRef);
        surfaceRef = nullptr;
    }
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_command(JNIEnv *env, jobject obj, jobjectArray args) {
    if (!mpv) return;
    jsize argc = env->GetArrayLength(args);
    auto **argv = new const char *[argc + 1];
    for (int i = 0; i < argc; i++) {
        auto str = (jstring)env->GetObjectArrayElement(args, i);
        argv[i] = str ? env->GetStringUTFChars(str, nullptr) : nullptr;
    }
    argv[argc] = nullptr;
    // Log del comando para diagnóstico
    if (argc > 0 && argv[0]) LOGI("command: %s %s", argv[0], argc > 1 && argv[1] ? argv[1] : "");
    int err = mpv_command(mpv, argv);
    if (err < 0) LOGE("command error: %s", mpv_error_string(err));
    for (int i = 0; i < argc; i++) {
        if (argv[i]) {
            auto str = (jstring)env->GetObjectArrayElement(args, i);
            env->ReleaseStringUTFChars(str, argv[i]);
        }
    }
    delete[] argv;
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_setOptionString(JNIEnv *env, jobject obj, jstring name, jstring value) {
    if (!mpv) return;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    const char *cvalue = env->GetStringUTFChars(value, nullptr);
    mpv_set_option_string(mpv, cname, cvalue);
    env->ReleaseStringUTFChars(name, cname);
    env->ReleaseStringUTFChars(value, cvalue);
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_setPropertyString(JNIEnv *env, jobject obj, jstring name, jstring value) {
    if (!mpv) return;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    const char *cvalue = env->GetStringUTFChars(value, nullptr);
    mpv_set_property_string(mpv, cname, cvalue);
    env->ReleaseStringUTFChars(name, cname);
    env->ReleaseStringUTFChars(value, cvalue);
}

JNIEXPORT jstring JNICALL
Java_com_p2pollo_mpv_MpvLib_getPropertyString(JNIEnv *env, jobject obj, jstring name) {
    if (!mpv) return nullptr;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    char *value = mpv_get_property_string(mpv, cname);
    env->ReleaseStringUTFChars(name, cname);
    if (!value) return nullptr;
    jstring result = env->NewStringUTF(value);
    mpv_free(value);
    return result;
}

JNIEXPORT jboolean JNICALL
Java_com_p2pollo_mpv_MpvLib_getPropertyBoolean(JNIEnv *env, jobject obj, jstring name) {
    if (!mpv) return JNI_FALSE;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    int value = 0;
    mpv_get_property(mpv, cname, MPV_FORMAT_FLAG, &value);
    env->ReleaseStringUTFChars(name, cname);
    return value ? JNI_TRUE : JNI_FALSE;
}

JNIEXPORT jdouble JNICALL
Java_com_p2pollo_mpv_MpvLib_getPropertyDouble(JNIEnv *env, jobject obj, jstring name) {
    if (!mpv) return 0.0;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    double value = 0.0;
    mpv_get_property(mpv, cname, MPV_FORMAT_DOUBLE, &value);
    env->ReleaseStringUTFChars(name, cname);
    return value;
}

JNIEXPORT jint JNICALL
Java_com_p2pollo_mpv_MpvLib_getPropertyInt(JNIEnv *env, jobject obj, jstring name) {
    if (!mpv) return 0;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    int64_t value = 0;
    mpv_get_property(mpv, cname, MPV_FORMAT_INT64, &value);
    env->ReleaseStringUTFChars(name, cname);
    return (jint)value;
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_setPropertyBoolean(JNIEnv *env, jobject obj, jstring name, jboolean value) {
    if (!mpv) return;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    int flag = value ? 1 : 0;
    mpv_set_property(mpv, cname, MPV_FORMAT_FLAG, &flag);
    env->ReleaseStringUTFChars(name, cname);
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_setPropertyDouble(JNIEnv *env, jobject obj, jstring name, jdouble value) {
    if (!mpv) return;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    mpv_set_property(mpv, cname, MPV_FORMAT_DOUBLE, &value);
    env->ReleaseStringUTFChars(name, cname);
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_setPropertyInt(JNIEnv *env, jobject obj, jstring name, jint value) {
    if (!mpv) return;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    int64_t v = value;
    mpv_set_property(mpv, cname, MPV_FORMAT_INT64, &v);
    env->ReleaseStringUTFChars(name, cname);
}

JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_observeProperty(JNIEnv *env, jobject obj, jstring name, jint format) {
    if (!mpv) return;
    const char *cname = env->GetStringUTFChars(name, nullptr);
    mpv_observe_property(mpv, 0, cname, (mpv_format)format);
    env->ReleaseStringUTFChars(name, cname);
}

// Procesar todos los eventos pendientes. Llamado desde Kotlin en hilo de eventos dedicado.
JNIEXPORT void JNICALL
Java_com_p2pollo_mpv_MpvLib_processPendingEvents(JNIEnv *env, jobject obj) {
    while (processOneMpvEvent(env) != 0) {}
}

} // extern "C"
