#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QIcon>
#include <QQuickWindow>
#include <QNetworkDiskCache>
#include <QStandardPaths>
#include "backend.h"
#include "mpvobject.h"

int main(int argc, char *argv[])
{
    // CRITICAL: Force OpenGL backend for QQuickFramebufferObject to work
    // Without this, Qt may use D3D11 or software rendering which doesn't support FBO rendering
    qputenv("QSG_RHI_BACKEND", "opengl");
    QQuickWindow::setGraphicsApi(QSGRendererInterface::OpenGL);

    QGuiApplication app(argc, argv);

    // Register MpvObject as QML type
    qmlRegisterType<MpvObject>("mpv", 1, 0, "MpvObject");

    // Set application metadata
    app.setOrganizationName("p2pollo");
    app.setApplicationName("p2pollo");
    app.setApplicationVersion("1.0.0");

    // Icono de la aplicación (barra de título y taskbar)
    app.setWindowIcon(QIcon(":/assets/images/output-estesi.png"));

    // Create backend instance
    Backend backend;

    // Create QML engine
    QQmlApplicationEngine engine;

    // Configure network cache for images (50 MB cache)
    QNetworkDiskCache *diskCache = new QNetworkDiskCache(&engine);
    QString cachePath = QStandardPaths::writableLocation(QStandardPaths::CacheLocation);
    diskCache->setCacheDirectory(cachePath + "/images");
    diskCache->setMaximumCacheSize(50 * 1024 * 1024); // 50 MB
    engine.networkAccessManager()->setCache(diskCache);

    // Expose backend to QML
    engine.rootContext()->setContextProperty("backend", &backend);

    // Load main QML file
    const QUrl url(QStringLiteral("qrc:/qml/main.qml"));

    QObject::connect(&engine, &QQmlApplicationEngine::objectCreated,
        &app, [url](QObject *obj, const QUrl &objUrl) {
            if (!obj && url == objUrl)
                QCoreApplication::exit(-1);
        }, Qt::QueuedConnection);

    engine.load(url);

    return app.exec();
}
