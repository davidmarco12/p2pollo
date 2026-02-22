#ifndef MPVOBJECT_H
#define MPVOBJECT_H

#include <QQuickFramebufferObject>
#include <QOpenGLFramebufferObject>
#include <mpv/client.h>
#include <mpv/render_gl.h>

class MpvRenderer;

/**
 * MpvObject: QML type que renderiza mpv usando OpenGL
 * Basado en mpv-examples/libmpv/qt_opengl
 */
class MpvObject : public QQuickFramebufferObject
{
    Q_OBJECT
    Q_PROPERTY(QString source READ source WRITE setSource NOTIFY sourceChanged)
    Q_PROPERTY(bool paused READ paused WRITE setPaused NOTIFY pausedChanged)
    Q_PROPERTY(double position READ position WRITE setPosition NOTIFY positionChanged)
    Q_PROPERTY(double duration READ duration NOTIFY durationChanged)
    Q_PROPERTY(int volume READ volume WRITE setVolume NOTIFY volumeChanged)
    Q_PROPERTY(bool subtitlesEnabled READ subtitlesEnabled WRITE setSubtitlesEnabled NOTIFY subtitlesEnabledChanged)

public:
    explicit MpvObject(QQuickItem *parent = nullptr);
    ~MpvObject() override;

    Renderer *createRenderer() const override;

protected:
    void geometryChange(const QRectF &newGeometry, const QRectF &oldGeometry) override;

    // Properties
    QString source() const { return m_source; }
    void setSource(const QString &source);

    bool paused() const { return m_paused; }
    void setPaused(bool paused);

    double position() const { return m_position; }
    void setPosition(double position);

    double duration() const { return m_duration; }

    int volume() const { return m_volume; }
    void setVolume(int volume);

    bool subtitlesEnabled() const { return m_subtitlesEnabled; }
    void setSubtitlesEnabled(bool enabled);

    // Commands
    Q_INVOKABLE void play();
    Q_INVOKABLE void pause();
    Q_INVOKABLE void stop();
    Q_INVOKABLE void seek(double seconds);
    Q_INVOKABLE void toggleSubtitles();
    Q_INVOKABLE void loadSubtitleFile(const QString &path);
    Q_INVOKABLE QVariantList getSubtitleTracks();
    Q_INVOKABLE void setSubtitleTrack(int trackId);

signals:
    void sourceChanged();
    void pausedChanged();
    void positionChanged();
    void durationChanged();
    void volumeChanged();
    void subtitlesEnabledChanged();

private slots:
    void handleMpvEvents();
    void doUpdate();

private:
    friend class MpvRenderer;

    mpv_handle *m_mpv;
    mpv_render_context *m_mpvGL;
    QString m_source;
    bool m_paused;
    double m_position;
    double m_duration;
    int m_volume;
    bool m_subtitlesEnabled;

    void setProperty(const QString &name, const QVariant &value);
    QVariant getProperty(const QString &name) const;
    void observeProperty(const QString &name);

    static void on_mpv_events(void *ctx);
    static void on_mpv_render_update(void *ctx);
};

/**
 * MpvRenderer: Renderiza frames de mpv en un FBO de Qt
 */
class MpvRenderer : public QQuickFramebufferObject::Renderer
{
public:
    MpvRenderer(MpvObject *obj);
    ~MpvRenderer() override;

    QOpenGLFramebufferObject *createFramebufferObject(const QSize &size) override;
    void render() override;

private:
    MpvObject *m_obj;
};

#endif // MPVOBJECT_H
