#include "mpvobject.h"
#include <QOpenGLContext>
#include <QOpenGLFramebufferObject>
#include <QQuickWindow>
#include <QDebug>

// Callback para eventos de mpv
static void wakeup(void *ctx)
{
    QMetaObject::invokeMethod((MpvObject*)ctx, "handleMpvEvents", Qt::QueuedConnection);
}

// Callback para actualización de render
static void on_update(void *ctx)
{
    QMetaObject::invokeMethod((MpvObject*)ctx, "doUpdate", Qt::QueuedConnection);
}

// OpenGL callback para mpv
static void *get_proc_address_mpv(void *ctx, const char *name)
{
    Q_UNUSED(ctx)
    QOpenGLContext *glctx = QOpenGLContext::currentContext();
    if (!glctx) return nullptr;
    return reinterpret_cast<void *>(glctx->getProcAddress(QByteArray(name)));
}

MpvObject::MpvObject(QQuickItem *parent)
    : QQuickFramebufferObject(parent)
    , m_mpv(nullptr)
    , m_mpvGL(nullptr)
    , m_paused(true)
    , m_position(0.0)
    , m_duration(0.0)
    , m_volume(100)
    , m_subtitlesEnabled(true)
    , m_bufferingForCache(false)
{
    qDebug() << "[MpvObject] Constructor - setting up FBO";

    // CRITICAL: Tell Qt to resize the FBO when item size changes
    setTextureFollowsItemSize(true);

    // Ensure item is visible and will be rendered
    setFlag(QQuickItem::ItemHasContents, true);

    qDebug() << "[MpvObject] FBO properties set, creating mpv instance";

    // Crear instancia de mpv
    m_mpv = mpv_create();
    if (!m_mpv) {
        qFatal("Could not create mpv instance");
        return;
    }

    // Configurar mpv
    mpv_set_option_string(m_mpv, "vo", "libmpv");
    mpv_set_option_string(m_mpv, "hwdec", "auto");
    mpv_set_option_string(m_mpv, "terminal", "yes");
    mpv_set_option_string(m_mpv, "msg-level", "all=v");

    // Inicializar mpv
    if (mpv_initialize(m_mpv) < 0) {
        qFatal("Could not initialize mpv");
        return;
    }

    // Observar propiedades
    observeProperty("time-pos");
    observeProperty("duration");
    observeProperty("pause");
    observeProperty("volume");
    // pausing-for-cache: true cuando mpv está esperando más datos del buffer
    // Se observa con MPV_FORMAT_FLAG porque es un booleano nativo de mpv
    mpv_observe_property(m_mpv, 0, "pausing-for-cache", MPV_FORMAT_FLAG);

    // Setup wakeup callback
    mpv_set_wakeup_callback(m_mpv, wakeup, this);

    // Wait for valid size before creating renderer
    auto checkAndCreateRenderer = [this]() {
        if (width() > 0 && height() > 0 && window()) {
            qDebug() << "[MpvObject] Item now has valid size:" << width() << "x" << height();
            qDebug() << "[MpvObject] Forcing update to create renderer";
            update();
        }
    };

    connect(this, &QQuickItem::widthChanged, this, checkAndCreateRenderer);
    connect(this, &QQuickItem::heightChanged, this, checkAndCreateRenderer);

    connect(this, &QQuickFramebufferObject::windowChanged,
            this, [this, checkAndCreateRenderer](QQuickWindow *win) {
                qDebug() << "[MpvObject] Window changed, win=" << win;
                if (win) {
                    qDebug() << "[MpvObject] Initial size:" << width() << "x" << height();
                    qDebug() << "[MpvObject] Connected to window's beforeRendering signal";
                    connect(win, &QQuickWindow::beforeRendering,
                            this, &MpvObject::doUpdate, Qt::DirectConnection);
                    // Check if size is already valid
                    checkAndCreateRenderer();
                }
            });
}

MpvObject::~MpvObject()
{
    if (m_mpvGL) {
        mpv_render_context_free(m_mpvGL);
    }
    if (m_mpv) {
        mpv_terminate_destroy(m_mpv);
    }
}

void MpvObject::geometryChange(const QRectF &newGeometry, const QRectF &oldGeometry)
{
    QQuickFramebufferObject::geometryChange(newGeometry, oldGeometry);
    qDebug() << "[MpvObject] geometryChange:" << oldGeometry << "=>" << newGeometry;

    if (newGeometry.width() > 0 && newGeometry.height() > 0) {
        qDebug() << "[MpvObject] Valid geometry, forcing update and polish";
        // Force Qt to recognize this item needs rendering
        polish();
        update();
    }
}

QQuickFramebufferObject::Renderer *MpvObject::createRenderer() const
{
    qDebug() << "[MpvObject] !!!! createRenderer() CALLED !!!!";
    qDebug() << "[MpvObject] Current size:" << width() << "x" << height();
    // Note: setPersistentOpenGLContext removed in Qt 6
    // Qt 6 handles this automatically
    MpvRenderer *renderer = new MpvRenderer(const_cast<MpvObject*>(this));
    qDebug() << "[MpvObject] MpvRenderer created, returning to Qt";
    return renderer;
}

void MpvObject::setSource(const QString &source)
{
    if (m_source != source) {
        m_source = source;

        qDebug() << "Loading source in mpv:" << source;

        // Cargar archivo/URL en mpv
        const char *cmd[] = {"loadfile", source.toUtf8().data(), nullptr};
        int result = mpv_command_async(m_mpv, 0, cmd);
        qDebug() << "mpv loadfile result:" << result;

        emit sourceChanged();
    }
}

void MpvObject::setPaused(bool paused)
{
    if (m_paused != paused) {
        m_paused = paused;  // Update immediately to prevent race condition
        setProperty("pause", paused);
        emit pausedChanged();
    }
}

void MpvObject::setPosition(double position)
{
    // Usar el comando seek en modo absoluto (más confiable que set time-pos)
    QByteArray posStr = QString::number(position, 'f', 3).toUtf8();
    const char *cmd[] = {"seek", posStr.data(), "absolute", nullptr};
    mpv_command_async(m_mpv, 0, cmd);
}

void MpvObject::setVolume(int volume)
{
    qDebug() << "[MpvObject] setVolume called:" << volume << "current:" << m_volume;
    if (m_volume != volume) {
        m_volume = volume;  // Update immediately
        qDebug() << "[MpvObject] Setting mpv volume to:" << volume;
        setProperty("volume", volume);
        emit volumeChanged();
    }
}

void MpvObject::play()
{
    setPaused(false);
}

void MpvObject::pause()
{
    setPaused(true);
}

void MpvObject::stop()
{
    const char *cmd[] = {"stop", nullptr};
    mpv_command_async(m_mpv, 0, cmd);
}

void MpvObject::seek(double seconds)
{
    QVariant args = QVariantList() << seconds << "relative";
    const char *cmd[] = {"seek", QString::number(seconds).toUtf8().data(), "relative", nullptr};
    mpv_command_async(m_mpv, 0, cmd);
}

void MpvObject::setSubtitlesEnabled(bool enabled)
{
    if (m_subtitlesEnabled != enabled) {
        m_subtitlesEnabled = enabled;
        qDebug() << "[MpvObject] Subtitles enabled:" << enabled;

        // mpv property "sid" controls subtitle track
        // "no" = disabled, "auto" = first available track
        if (enabled) {
            mpv_set_property_string(m_mpv, "sid", "auto");
        } else {
            mpv_set_property_string(m_mpv, "sid", "no");
        }

        emit subtitlesEnabledChanged();
    }
}

void MpvObject::toggleSubtitles()
{
    setSubtitlesEnabled(!m_subtitlesEnabled);
}

void MpvObject::loadSubtitleFile(const QString &path)
{
    if (path.isEmpty() || !m_mpv) return;

    qDebug() << "[MpvObject] Loading subtitle file:" << path;
    QByteArray pathUtf8 = path.toUtf8();
    const char *cmd[] = {"sub-add", pathUtf8.data(), nullptr};
    mpv_command_async(m_mpv, 0, cmd);

    // Enable subtitles after loading
    setSubtitlesEnabled(true);
}

QVariantList MpvObject::getSubtitleTracks()
{
    QVariantList tracks;
    if (!m_mpv) return tracks;

    // Get track-list property from mpv
    mpv_node node;
    if (mpv_get_property(m_mpv, "track-list", MPV_FORMAT_NODE, &node) < 0) {
        qWarning() << "[MpvObject] Failed to get track-list";
        return tracks;
    }

    // track-list is an MPV_FORMAT_NODE_ARRAY
    if (node.format == MPV_FORMAT_NODE_ARRAY) {
        mpv_node_list *list = node.u.list;
        for (int i = 0; i < list->num; i++) {
            mpv_node *item = &list->values[i];
            if (item->format != MPV_FORMAT_NODE_MAP) continue;

            mpv_node_list *map = item->u.list;
            QVariantMap track;
            QString type;

            // Parse track properties
            for (int j = 0; j < map->num; j++) {
                QString key = QString::fromUtf8(map->keys[j]);
                mpv_node *value = &map->values[j];

                if (key == "type" && value->format == MPV_FORMAT_STRING) {
                    type = QString::fromUtf8(value->u.string);
                } else if (key == "id" && value->format == MPV_FORMAT_INT64) {
                    track["id"] = (int)value->u.int64;
                } else if (key == "lang" && value->format == MPV_FORMAT_STRING) {
                    track["lang"] = QString::fromUtf8(value->u.string);
                } else if (key == "title" && value->format == MPV_FORMAT_STRING) {
                    track["title"] = QString::fromUtf8(value->u.string);
                }
            }

            // Only add subtitle tracks
            if (type == "sub") {
                tracks.append(track);
            }
        }
    }

    mpv_free_node_contents(&node);
    return tracks;
}

void MpvObject::setSubtitleTrack(int trackId)
{
    if (!m_mpv) return;

    qDebug() << "[MpvObject] Setting subtitle track to:" << trackId;

    if (trackId == 0) {
        // 0 means disable subtitles
        mpv_set_property_string(m_mpv, "sid", "no");
        m_subtitlesEnabled = false;
    } else {
        // Set specific track ID
        int64_t id = trackId;
        mpv_set_property(m_mpv, "sid", MPV_FORMAT_INT64, &id);
        m_subtitlesEnabled = true;
    }

    emit subtitlesEnabledChanged();
}

void MpvObject::handleMpvEvents()
{
    while (m_mpv) {
        mpv_event *event = mpv_wait_event(m_mpv, 0);
        if (event->event_id == MPV_EVENT_NONE) {
            break;
        }

        qDebug() << "MPV Event:" << mpv_event_name(event->event_id);

        switch (event->event_id) {
        case MPV_EVENT_FILE_LOADED:
            qDebug() << "File loaded successfully";
            break;
        case MPV_EVENT_START_FILE:
            qDebug() << "Starting file playback";
            break;
        case MPV_EVENT_END_FILE: {
            mpv_event_end_file *ef = (mpv_event_end_file *)event->data;
            qDebug() << "End file, reason:" << ef->reason << "error:" << ef->error;
            break;
        }
        case MPV_EVENT_PLAYBACK_RESTART:
            qDebug() << "Playback restarted";
            break;
        case MPV_EVENT_PROPERTY_CHANGE: {
            mpv_event_property *prop = (mpv_event_property *)event->data;
            if (strcmp(prop->name, "time-pos") == 0) {
                if (prop->format == MPV_FORMAT_DOUBLE) {
                    double pos = *(double *)prop->data;
                    if (m_position != pos) {
                        m_position = pos;
                        emit positionChanged();
                    }
                }
            } else if (strcmp(prop->name, "duration") == 0) {
                if (prop->format == MPV_FORMAT_DOUBLE) {
                    double dur = *(double *)prop->data;
                    if (m_duration != dur) {
                        m_duration = dur;
                        emit durationChanged();
                    }
                }
            } else if (strcmp(prop->name, "pause") == 0) {
                if (prop->format == MPV_FORMAT_FLAG) {
                    bool paused = *(int *)prop->data;
                    if (m_paused != paused) {
                        m_paused = paused;
                        emit pausedChanged();
                    }
                }
            } else if (strcmp(prop->name, "volume") == 0) {
                if (prop->format == MPV_FORMAT_DOUBLE) {
                    int vol = (int)(*(double *)prop->data);
                    if (m_volume != vol) {
                        m_volume = vol;
                        emit volumeChanged();
                    }
                }
            } else if (strcmp(prop->name, "pausing-for-cache") == 0) {
                if (prop->format == MPV_FORMAT_FLAG) {
                    bool buffering = *(int *)prop->data;
                    if (m_bufferingForCache != buffering) {
                        m_bufferingForCache = buffering;
                        emit bufferingForCacheChanged();
                    }
                }
            }
            break;
        }
        case MPV_EVENT_LOG_MESSAGE: {
            mpv_event_log_message *msg = (mpv_event_log_message *)event->data;
            qDebug() << "[mpv]" << msg->prefix << msg->level << msg->text;
            break;
        }
        default:
            break;
        }
    }
}

void MpvObject::doUpdate()
{
    // CRITICAL: For QQuickFramebufferObject, we need to call update() on the item itself
    // not window()->update(), to trigger render() calls
    update();

    static int updateCount = 0;
    if (updateCount++ % 60 == 0) {  // Log every 60 updates
        qDebug() << "[doUpdate] Triggered FBO update" << updateCount;
    }
}

void MpvObject::setProperty(const QString &name, const QVariant &value)
{
    if (!m_mpv) return;

    QByteArray nameUtf8 = name.toUtf8();

    if (value.metaType().id() == QMetaType::QString) {
        QByteArray valueUtf8 = value.toString().toUtf8();
        mpv_set_property_string(m_mpv, nameUtf8.data(), valueUtf8.data());
    } else if (value.metaType().id() == QMetaType::Bool) {
        int flag = value.toBool() ? 1 : 0;
        mpv_set_property(m_mpv, nameUtf8.data(), MPV_FORMAT_FLAG, &flag);
    } else if (value.canConvert<double>()) {
        double d = value.toDouble();
        mpv_set_property(m_mpv, nameUtf8.data(), MPV_FORMAT_DOUBLE, &d);
    } else if (value.canConvert<int>()) {
        int64_t i = value.toInt();
        mpv_set_property(m_mpv, nameUtf8.data(), MPV_FORMAT_INT64, &i);
    }
}

QVariant MpvObject::getProperty(const QString &name) const
{
    if (!m_mpv) return QVariant();

    QByteArray nameUtf8 = name.toUtf8();
    char *value = mpv_get_property_string(m_mpv, nameUtf8.data());
    if (!value) return QVariant();

    QVariant result = QString::fromUtf8(value);
    mpv_free(value);
    return result;
}

void MpvObject::observeProperty(const QString &name)
{
    if (!m_mpv) return;
    mpv_observe_property(m_mpv, 0, name.toUtf8().data(), MPV_FORMAT_DOUBLE);
}

// --- MpvRenderer ---

MpvRenderer::MpvRenderer(MpvObject *obj)
    : m_obj(obj)
{
    qDebug() << "[MpvRenderer] Constructor called";

    // Inicializar render context de mpv (OpenGL)
    if (!m_obj->m_mpvGL) {
        qDebug() << "[MpvRenderer] Creating mpv OpenGL render context...";
        mpv_opengl_init_params gl_init_params{get_proc_address_mpv, nullptr};
        mpv_render_param params[]{
            {MPV_RENDER_PARAM_API_TYPE, const_cast<char *>(MPV_RENDER_API_TYPE_OPENGL)},
            {MPV_RENDER_PARAM_OPENGL_INIT_PARAMS, &gl_init_params},
            {MPV_RENDER_PARAM_INVALID, nullptr}
        };

        if (mpv_render_context_create(&m_obj->m_mpvGL, m_obj->m_mpv, params) < 0) {
            qFatal("Failed to initialize mpv OpenGL context");
        }

        qDebug() << "[MpvRenderer] OpenGL context created successfully";
        mpv_render_context_set_update_callback(m_obj->m_mpvGL, on_update, m_obj);
        qDebug() << "[MpvRenderer] Update callback set";
    } else {
        qDebug() << "[MpvRenderer] OpenGL context already exists";
    }
}

MpvRenderer::~MpvRenderer()
{
}

QOpenGLFramebufferObject *MpvRenderer::createFramebufferObject(const QSize &size)
{
    return QQuickFramebufferObject::Renderer::createFramebufferObject(size);
}

void MpvRenderer::render()
{
    if (!m_obj->m_mpvGL) {
        qDebug() << "[Render] No mpvGL context!";
        return;
    }

    QOpenGLFramebufferObject *fbo = framebufferObject();
    if (!fbo) {
        qDebug() << "[Render] No FBO!";
        return;
    }

    mpv_opengl_fbo mpfbo{
        static_cast<int>(fbo->handle()),
        fbo->width(),
        fbo->height(),
        0
    };

    int flip_y{0};  // 0 = don't flip (Qt FBO is already in correct orientation)
    mpv_render_param params[] = {
        {MPV_RENDER_PARAM_OPENGL_FBO, &mpfbo},
        {MPV_RENDER_PARAM_FLIP_Y, &flip_y},
        {MPV_RENDER_PARAM_INVALID, nullptr}
    };

    // Render frame
    mpv_render_context_render(m_obj->m_mpvGL, params);

    static int frameCount = 0;
    if (frameCount++ % 60 == 0) {  // Log every 60 frames
        qDebug() << "[Render] Rendered frame" << frameCount << "size:" << fbo->width() << "x" << fbo->height();
    }
}
