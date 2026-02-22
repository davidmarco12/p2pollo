#include "backend.h"
#include <QNetworkRequest>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QDebug>

Backend::Backend(QObject *parent)
    : QObject(parent)
    , m_networkManager(new QNetworkAccessManager(this))
    , m_apiBaseUrl("http://127.0.0.1:9876")
    , m_isLoading(false)
    , m_isStreamReady(false)
    , m_streamPathTimer(new QTimer(this))
    , m_progressTimer(new QTimer(this))
{
    connect(m_networkManager, &QNetworkAccessManager::finished,
            this, &Backend::handleNetworkReply);

    // Configure polling timers
    m_streamPathTimer->setInterval(1000); // 1 second
    connect(m_streamPathTimer, &QTimer::timeout, this, &Backend::pollStreamPath);

    m_progressTimer->setInterval(1000);
    connect(m_progressTimer, &QTimer::timeout, this, &Backend::pollProgress);
}

Backend::~Backend()
{
    stopStreamPathPolling();
    stopProgressPolling();
}

void Backend::setApiBaseUrl(const QString &url)
{
    if (m_apiBaseUrl != url) {
        m_apiBaseUrl = url;
        emit apiBaseUrlChanged();
    }
}

// --- API Methods ---

void Backend::getPopularMovies()
{
    sendGetRequest("/api/popular", "popularMovies");
}

void Backend::searchMovies(const QString &query)
{
    sendGetRequest("/api/search?q=" + QUrl::toPercentEncoding(query), "searchMovies");
}

void Backend::getMovieDetails(const QJsonObject &movie)
{
    sendPostRequest("/api/movie/details", movie, "movieDetails");
}

void Backend::playMagnet(const QString &magnetLink, int fileIndex)
{
    QJsonObject data;
    data["magnetLink"] = magnetLink;
    data["fileIndex"] = fileIndex;
    sendPostRequest("/api/play", data, "playMagnet");
}

void Backend::stopStream()
{
    sendPostRequest("/api/stop", QJsonObject(), "stopStream");
}

void Backend::getProgress()
{
    sendGetRequest("/api/progress", "progress");
}

// --- Polling ---

void Backend::startStreamPathPolling()
{
    qDebug() << "[Poll] Starting stream path polling (interval: 1s)";
    m_streamPathTimer->start();
}

void Backend::stopStreamPathPolling()
{
    qDebug() << "[Poll] Stopping stream path polling";
    m_streamPathTimer->stop();
}

void Backend::startProgressPolling()
{
    m_progressTimer->start();
}

void Backend::stopProgressPolling()
{
    m_progressTimer->stop();
}

void Backend::pollStreamPath()
{
    qDebug() << "[Poll] Polling stream path...";
    sendGetRequest("/api/stream-path", "streamPath");
}

void Backend::pollProgress()
{
    sendGetRequest("/api/progress", "progress");
}

// --- HTTP Helpers ---

void Backend::sendGetRequest(const QString &endpoint, const QString &context)
{
    QUrl url(m_apiBaseUrl + endpoint);
    QNetworkRequest request(url);
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");

    QNetworkReply *reply = m_networkManager->get(request);
    m_replyContext[reply] = context;

    m_isLoading = true;
    emit isLoadingChanged();
}

void Backend::sendPostRequest(const QString &endpoint, const QJsonObject &data, const QString &context)
{
    QUrl url(m_apiBaseUrl + endpoint);
    QNetworkRequest request(url);
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");

    QJsonDocument doc(data);
    QByteArray body = doc.toJson();

    QNetworkReply *reply = m_networkManager->post(request, body);
    m_replyContext[reply] = context;

    m_isLoading = true;
    emit isLoadingChanged();
}

void Backend::handleNetworkReply(QNetworkReply *reply)
{
    reply->deleteLater();

    m_isLoading = false;
    emit isLoadingChanged();

    QString context = m_replyContext.take(reply);

    if (reply->error() != QNetworkReply::NoError) {
        QString error = reply->errorString();
        qWarning() << "Network error:" << error << "for context:" << context;
        emit errorOccurred(error);
        return;
    }

    QByteArray responseData = reply->readAll();
    QJsonDocument doc = QJsonDocument::fromJson(responseData);

    // Handle different contexts
    if (context == "popularMovies" || context == "searchMovies") {
        if (doc.isArray()) {
            QJsonArray movies = doc.array();
            if (context == "popularMovies") {
                emit popularMoviesReceived(movies);
            } else {
                emit searchResultsReceived(movies);
            }
        }
    }
    else if (context == "movieDetails") {
        if (doc.isObject()) {
            emit movieDetailsReceived(doc.object());
        }
    }
    else if (context == "playMagnet") {
        qDebug() << "[playMagnet] Stream started, beginning polling";
        emit streamStarted();
        // Auto-start polling when stream starts
        startStreamPathPolling();
        startProgressPolling();
    }
    else if (context == "stopStream") {
        emit streamStopped();
        stopStreamPathPolling();
        stopProgressPolling();
        m_streamPath.clear();
        m_isStreamReady = false;
        emit streamPathChanged();
        emit isStreamReadyChanged();
    }
    else if (context == "streamPath") {
        if (doc.isObject()) {
            QJsonObject obj = doc.object();
            QString path = obj["path"].toString();
            bool ready = obj["ready"].toBool();

            qDebug() << "[streamPath] Received JSON - path:" << path << "ready:" << ready;
            qDebug() << "[streamPath] Current state - m_streamPath:" << m_streamPath << "m_isStreamReady:" << m_isStreamReady;

            if (m_streamPath != path) {
                m_streamPath = path;
                qDebug() << "[streamPath] Path changed, emitting streamPathChanged()";
                emit streamPathChanged();
            }

            if (m_isStreamReady != ready) {
                m_isStreamReady = ready;
                qDebug() << "[streamPath] Ready changed to:" << ready << ", emitting isStreamReadyChanged()";
                emit isStreamReadyChanged();

                // When stream is ready, stop path polling
                if (ready) {
                    qDebug() << "[streamPath] Stream ready, stopping path polling";
                    stopStreamPathPolling();
                }
            } else {
                qDebug() << "[streamPath] Ready unchanged, no signal emitted";
            }
        } else {
            qWarning() << "[streamPath] Response is not a JSON object!";
        }
    }
    else if (context == "progress") {
        if (doc.isObject()) {
            emit progressReceived(doc.object());
        }
    }
}
