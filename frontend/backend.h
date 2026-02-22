#ifndef BACKEND_H
#define BACKEND_H

#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QTimer>

/**
 * Backend: Clase C++ que maneja la comunicación HTTP con el API Go.
 * Expuesta a QML para que los componentes puedan consumir los endpoints.
 */
class Backend : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString apiBaseUrl READ apiBaseUrl WRITE setApiBaseUrl NOTIFY apiBaseUrlChanged)
    Q_PROPERTY(bool isLoading READ isLoading NOTIFY isLoadingChanged)
    Q_PROPERTY(QString streamPath READ streamPath NOTIFY streamPathChanged)
    Q_PROPERTY(bool isStreamReady READ isStreamReady NOTIFY isStreamReadyChanged)

public:
    explicit Backend(QObject *parent = nullptr);
    ~Backend();

    // Getters
    QString apiBaseUrl() const { return m_apiBaseUrl; }
    bool isLoading() const { return m_isLoading; }
    QString streamPath() const { return m_streamPath; }
    bool isStreamReady() const { return m_isStreamReady; }

    // Setters
    void setApiBaseUrl(const QString &url);

    // API Methods (invocables from QML)
    Q_INVOKABLE void getPopularMovies();
    Q_INVOKABLE void searchMovies(const QString &query);
    Q_INVOKABLE void getMovieDetails(const QJsonObject &movie);
    Q_INVOKABLE void playMagnet(const QString &magnetLink, int fileIndex = -1);
    Q_INVOKABLE void stopStream();
    Q_INVOKABLE void getProgress();

    // Helpers
    Q_INVOKABLE void startStreamPathPolling();
    Q_INVOKABLE void stopStreamPathPolling();
    Q_INVOKABLE void startProgressPolling();
    Q_INVOKABLE void stopProgressPolling();

signals:
    // Property change signals
    void apiBaseUrlChanged();
    void isLoadingChanged();
    void streamPathChanged();
    void isStreamReadyChanged();

    // API response signals
    void popularMoviesReceived(const QJsonArray &movies);
    void searchResultsReceived(const QJsonArray &movies);
    void movieDetailsReceived(const QJsonObject &details);
    void streamStarted();
    void streamStopped();
    void progressReceived(const QJsonObject &progress);

    // Error signals
    void errorOccurred(const QString &error);

private slots:
    void handleNetworkReply(QNetworkReply *reply);
    void pollStreamPath();
    void pollProgress();

private:
    // HTTP helpers
    void sendGetRequest(const QString &endpoint, const QString &context);
    void sendPostRequest(const QString &endpoint, const QJsonObject &data, const QString &context);

    QNetworkAccessManager *m_networkManager;
    QString m_apiBaseUrl;
    bool m_isLoading;
    QString m_streamPath;
    bool m_isStreamReady;

    // Polling timers
    QTimer *m_streamPathTimer;
    QTimer *m_progressTimer;

    // Context tracking for async requests
    QHash<QNetworkReply*, QString> m_replyContext;
};

#endif // BACKEND_H
