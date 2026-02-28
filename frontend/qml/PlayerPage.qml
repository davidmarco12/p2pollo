import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import mpv 1.0

Item {
    id: root

    property string magnetLink: ""
    property var progress: null
    property bool controlsVisible: true

    signal backRequested()

    Timer {
        id: hideControlsTimer
        interval: 3000
        repeat: false
        onTriggered: {
            if (!seekSlider.pressed && !volumeSlider.pressed) {
                controlsVisible = false
            }
        }
    }

    Rectangle {
        anchors.fill: parent
        color: "#000"

        // ── Cursor / controles show/hide ─────────────────────────────────
        MouseArea {
            anchors.fill: parent
            hoverEnabled: true
            propagateComposedEvents: true
            acceptedButtons: Qt.NoButton
            cursorShape: controlsVisible ? Qt.ArrowCursor : Qt.BlankCursor

            onPositionChanged: {
                if (!controlsVisible) controlsVisible = true
                hideControlsTimer.restart()
            }
            onEntered: {
                controlsVisible = true
                hideControlsTimer.restart()
            }
        }

        // ── MPV (full screen) ────────────────────────────────────────────
        MpvObject {
            id: mpvPlayer
            anchors.fill: parent

            Component.onCompleted: console.log("MpvObject created")
            onPausedChanged: console.log("Paused:", paused)

            onPositionChanged: {
                if (!seekSlider.pressed)
                    seekSlider.value = position
            }
            onVolumeChanged: {
                if (!volumeSlider.pressed)
                    volumeSlider.value = volume
            }
            onDurationChanged: console.log("Duration:", duration)
        }

        // ── Loading overlay ──────────────────────────────────────────────
        Rectangle {
            anchors.fill: parent
            color: "#000"
            visible: !backend.isStreamReady
            z: 10

            ColumnLayout {
                anchors.centerIn: parent
                spacing: 16

                BusyIndicator {
                    Layout.alignment: Qt.AlignHCenter
                    running: true
                }

                Text {
                    Layout.alignment: Qt.AlignHCenter
                    text: {
                        if (!progress) return "Preparando stream..."
                        if (progress.preparing) return "Conectando al torrent..."
                        if (progress.error) return "Error: " + progress.error
                        return "Descargando buffer inicial... " + progress.percent + "%"
                    }
                    font.pixelSize: 16
                    color: "#f5f3f0"
                }

                Text {
                    Layout.alignment: Qt.AlignHCenter
                    text: progress && progress.peers > 0 ? progress.peers + " peers conectados" : ""
                    font.pixelSize: 14
                    color: "#8a837c"
                    visible: text !== ""
                }

                Text {
                    Layout.alignment: Qt.AlignHCenter
                    text: progress && progress.speedMBps > 0 ? progress.speedMBps.toFixed(2) + " MB/s" : ""
                    font.pixelSize: 14
                    color: "#8a837c"
                    visible: text !== ""
                }
            }
        }

        // ── Buffering mid-playback overlay ───────────────────────────────
        Rectangle {
            anchors.fill: parent
            color: "#99000000"
            visible: backend.isStreamReady && mpvPlayer.bufferingForCache
            z: 9

            ColumnLayout {
                anchors.centerIn: parent
                spacing: 12

                BusyIndicator {
                    Layout.alignment: Qt.AlignHCenter
                    running: parent.parent.visible
                }

                Text {
                    Layout.alignment: Qt.AlignHCenter
                    text: "Descargando..."
                    font.pixelSize: 15
                    color: "#f5f3f0"
                }

                Text {
                    Layout.alignment: Qt.AlignHCenter
                    text: progress && progress.speedMBps > 0 ? progress.speedMBps.toFixed(2) + " MB/s" : ""
                    font.pixelSize: 13
                    color: "#8a837c"
                    visible: text !== ""
                }
            }
        }

        // ── Controls overlay (bottom, sobre el video) ────────────────────
        Item {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: controlsVisible ? 110 : 0
            visible: backend.isStreamReady
            z: 5
            clip: true

            Behavior on height {
                NumberAnimation { duration: 200; easing.type: Easing.InOutQuad }
            }

            // Degradado negro transparente → sólido
            Rectangle {
                anchors.fill: parent
                gradient: Gradient {
                    orientation: Gradient.Vertical
                    GradientStop { position: 0.0; color: "#00000000" }
                    GradientStop { position: 0.4; color: "#aa000000" }
                    GradientStop { position: 1.0; color: "#ee000000" }
                }
            }

            ColumnLayout {
                anchors.fill: parent
                anchors.leftMargin: 16
                anchors.rightMargin: 16
                anchors.bottomMargin: 10
                anchors.topMargin: 10
                spacing: 4

                // ── Seek slider ──────────────────────────────────────────
                Slider {
                    id: seekSlider
                    Layout.fillWidth: true
                    from: 0
                    to: mpvPlayer.duration > 0 ? mpvPlayer.duration : 1
                    value: 0

                    onMoved: {
                        if (mpvPlayer.duration > 0)
                            mpvPlayer.position = value
                    }

                    background: Rectangle {
                        x: seekSlider.leftPadding
                        y: seekSlider.topPadding + seekSlider.availableHeight / 2 - height / 2
                        implicitHeight: 20
                        width: seekSlider.availableWidth
                        height: 5
                        radius: 2.5
                        color: "#33ffffff"

                        // Buffer descargado
                        Rectangle {
                            width: {
                                if (!progress || progress.totalSize <= 0) return 0
                                return Math.min(progress.headWritten / progress.totalSize, 1.0) * parent.width
                            }
                            height: parent.height
                            color: "#66ffffff"
                            radius: 2.5
                        }

                        // Posición actual
                        Rectangle {
                            width: seekSlider.visualPosition * parent.width
                            height: parent.height
                            color: "#ff6b35"
                            radius: 2.5
                        }
                    }

                    handle: Rectangle {
                        x: seekSlider.leftPadding + seekSlider.visualPosition * (seekSlider.availableWidth - width)
                        y: seekSlider.topPadding + seekSlider.availableHeight / 2 - height / 2
                        width: 14; height: 14; radius: 7
                        color: "#ff6b35"
                        border.color: "#fff"
                        border.width: 2
                    }
                }

                // ── Fila info: buffer % y tiempo ─────────────────────────
                RowLayout {
                    Layout.fillWidth: true
                    Layout.topMargin: -2

                    Text {
                        text: {
                            if (!progress || progress.totalSize <= 0) return ""
                            return "Buffer: " + Math.round(progress.headWritten / progress.totalSize * 100) + "%"
                        }
                        font.pixelSize: 11
                        color: "#99ffffff"
                        visible: text !== ""
                    }

                    Item { Layout.fillWidth: true }

                    Text {
                        text: formatTime(mpvPlayer.position) + " / " + formatTime(mpvPlayer.duration)
                        font.pixelSize: 12
                        font.weight: Font.Medium
                        color: "#f5f3f0"
                    }
                }

                // ── Botones de control ───────────────────────────────────
                RowLayout {
                    Layout.fillWidth: true
                    spacing: 4

                    // Volver
                    Rectangle {
                        width: 36; height: 36; radius: 4
                        color: backMa.containsMouse ? "#33ffffff" : "transparent"
                        Text {
                            anchors.centerIn: parent
                            text: "←"; font.pixelSize: 18; color: "#f5f3f0"
                        }
                        MouseArea {
                            id: backMa; anchors.fill: parent
                            hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                            onClicked: root.backRequested()
                        }
                    }

                    // Retroceder 10s
                    Rectangle {
                        width: 36; height: 36; radius: 4
                        color: rewindMa.containsMouse ? "#33ffffff" : "transparent"
                        Canvas {
                            anchors.centerIn: parent
                            width: 22; height: 16
                            onPaint: {
                                var ctx = getContext("2d")
                                ctx.clearRect(0, 0, width, height)
                                ctx.fillStyle = "#f5f3f0"
                                // Barra vertical izquierda
                                ctx.fillRect(0, 0, 2, 16)
                                // Triángulo 1 apuntando izquierda (punta en x=3, base en x=11)
                                ctx.beginPath()
                                ctx.moveTo(3, 8)
                                ctx.lineTo(11, 0)
                                ctx.lineTo(11, 16)
                                ctx.closePath()
                                ctx.fill()
                                // Triángulo 2 apuntando izquierda (punta en x=12, base en x=22)
                                ctx.beginPath()
                                ctx.moveTo(12, 8)
                                ctx.lineTo(22, 0)
                                ctx.lineTo(22, 16)
                                ctx.closePath()
                                ctx.fill()
                            }
                        }
                        MouseArea {
                            id: rewindMa; anchors.fill: parent
                            hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                            onClicked: mpvPlayer.seek(-10)
                        }
                    }

                    // Play / Pause
                    Rectangle {
                        width: 44; height: 44; radius: 22
                        color: playMa.containsMouse ? "#ff8555" : "#ff6b35"
                        Canvas {
                            id: playPauseCanvas
                            anchors.centerIn: parent
                            width: 18; height: 18
                            property bool isPaused: mpvPlayer.paused
                            onIsPausedChanged: requestPaint()
                            onPaint: {
                                var ctx = getContext("2d")
                                ctx.clearRect(0, 0, width, height)
                                ctx.fillStyle = "#0a0908"
                                if (isPaused) {
                                    // Triángulo play apuntando derecha
                                    ctx.beginPath()
                                    ctx.moveTo(width, height / 2)
                                    ctx.lineTo(2, 0)
                                    ctx.lineTo(2, height)
                                    ctx.closePath()
                                    ctx.fill()
                                } else {
                                    // Dos barras de pausa
                                    ctx.fillRect(1, 0, 5, height)
                                    ctx.fillRect(width - 6, 0, 5, height)
                                }
                            }
                        }
                        MouseArea {
                            id: playMa; anchors.fill: parent
                            hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                            onClicked: mpvPlayer.paused = !mpvPlayer.paused
                        }
                    }

                    // Adelantar 30s
                    Rectangle {
                        width: 36; height: 36; radius: 4
                        color: fwdMa.containsMouse ? "#33ffffff" : "transparent"
                        Canvas {
                            anchors.centerIn: parent
                            width: 22; height: 16
                            onPaint: {
                                var ctx = getContext("2d")
                                ctx.clearRect(0, 0, width, height)
                                ctx.fillStyle = "#f5f3f0"
                                // Triángulo 1 apuntando derecha (punta en x=10, base en x=0)
                                ctx.beginPath()
                                ctx.moveTo(10, 8)
                                ctx.lineTo(0, 0)
                                ctx.lineTo(0, 16)
                                ctx.closePath()
                                ctx.fill()
                                // Triángulo 2 apuntando derecha (punta en x=19, base en x=11)
                                ctx.beginPath()
                                ctx.moveTo(19, 8)
                                ctx.lineTo(11, 0)
                                ctx.lineTo(11, 16)
                                ctx.closePath()
                                ctx.fill()
                                // Barra vertical derecha
                                ctx.fillRect(20, 0, 2, 16)
                            }
                        }
                        MouseArea {
                            id: fwdMa; anchors.fill: parent
                            hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                            onClicked: mpvPlayer.seek(30)
                        }
                    }

                    Item { Layout.fillWidth: true }

                    // Volumen
                    RowLayout {
                        spacing: 6

                        Text { text: "🔊"; font.pixelSize: 14; color: "#f5f3f0" }

                        Slider {
                            id: volumeSlider
                            from: 0; to: 100; value: 50
                            Layout.preferredWidth: 90

                            onMoved: mpvPlayer.volume = Math.round(value)

                            background: Rectangle {
                                x: volumeSlider.leftPadding
                                y: volumeSlider.topPadding + volumeSlider.availableHeight / 2 - height / 2
                                width: volumeSlider.availableWidth
                                height: 4; radius: 2
                                color: "#33ffffff"

                                Rectangle {
                                    width: volumeSlider.visualPosition * parent.width
                                    height: parent.height; color: "#ffffff"; radius: 2
                                }
                            }

                            handle: Rectangle {
                                x: volumeSlider.leftPadding + volumeSlider.visualPosition * (volumeSlider.availableWidth - width)
                                y: volumeSlider.topPadding + volumeSlider.availableHeight / 2 - height / 2
                                width: 12; height: 12; radius: 6; color: "#fff"
                            }
                        }

                        Text {
                            text: Math.round(volumeSlider.value) + "%"
                            font.pixelSize: 11; color: "#99ffffff"
                            Layout.preferredWidth: 34
                        }
                    }

                    // Subtítulos (CC) — Button necesario para el Menu popup
                    Button {
                        id: subtitlesButton
                        text: "CC"
                        font.pixelSize: 12
                        font.bold: true
                        property int currentSubTrack: -1

                        onClicked: subtitlesMenu.popup()

                        background: Rectangle {
                            color: subtitlesButton.hovered ? "#33ffffff" : "transparent"
                            border.color: mpvPlayer.subtitlesEnabled ? "#ff6b35" : "#66ffffff"
                            border.width: 1; radius: 4
                            implicitWidth: 34; implicitHeight: 28
                        }
                        contentItem: Text {
                            text: parent.text
                            color: mpvPlayer.subtitlesEnabled ? "#ff6b35" : "#f5f3f0"
                            font: parent.font
                            horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                        }

                        Menu {
                            id: subtitlesMenu
                            y: -height

                            onAboutToShow: {
                                while (subtitlesMenu.count > 2)
                                    subtitlesMenu.removeItem(subtitlesMenu.itemAt(2))
                                var tracks = mpvPlayer.getSubtitleTracks()
                                for (var i = 0; i < tracks.length; i++) {
                                    var track = tracks[i]
                                    var label = track.lang || "Unknown"
                                    if (track.title) label += " - " + track.title
                                    var item = menuItemComponent.createObject(subtitlesMenu, { text: label, trackId: track.id })
                                    subtitlesMenu.addItem(item)
                                }
                            }

                            MenuItem {
                                text: "Desactivar"
                                indicator: Rectangle {
                                    width: 8; height: 8; radius: 4
                                    anchors.verticalCenter: parent ? parent.verticalCenter : undefined
                                    color: subtitlesButton.currentSubTrack === 0 ? "#ff6b35" : "transparent"
                                    border.color: subtitlesButton.currentSubTrack === 0 ? "#ff6b35" : "#555"
                                    border.width: 1
                                }
                                onTriggered: {
                                    mpvPlayer.setSubtitleTrack(0)
                                    subtitlesButton.currentSubTrack = 0
                                }
                            }
                            MenuSeparator {}
                        }

                        Component {
                            id: menuItemComponent
                            MenuItem {
                                property int trackId: 0
                                indicator: Rectangle {
                                    width: 8; height: 8; radius: 4
                                    anchors.verticalCenter: parent ? parent.verticalCenter : undefined
                                    color: trackId === subtitlesButton.currentSubTrack ? "#ff6b35" : "transparent"
                                    border.color: trackId === subtitlesButton.currentSubTrack ? "#ff6b35" : "#555"
                                    border.width: 1
                                }
                                onTriggered: {
                                    mpvPlayer.setSubtitleTrack(trackId)
                                    subtitlesButton.currentSubTrack = trackId
                                }
                            }
                        }
                    }

                    // Fullscreen
                    Rectangle {
                        width: 36; height: 36; radius: 4
                        color: fsMa.containsMouse ? "#33ffffff" : "transparent"
                        Text {
                            anchors.centerIn: parent
                            text: root.Window.window && root.Window.window.visibility === Window.FullScreen ? "🗗" : "🗖"
                            font.pixelSize: 16; color: "#f5f3f0"
                        }
                        MouseArea {
                            id: fsMa; anchors.fill: parent
                            hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                            onClicked: {
                                if (root.Window.window) {
                                    if (root.Window.window.visibility === Window.FullScreen) {
                                        root.Window.window.showNormal()
                                        controlsVisible = true
                                    } else {
                                        root.Window.window.showFullScreen()
                                        hideControlsTimer.restart()
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    function formatTime(seconds) {
        if (!seconds || seconds < 0) return "0:00"
        var h = Math.floor(seconds / 3600)
        var m = Math.floor((seconds % 3600) / 60)
        var s = Math.floor(seconds % 60)
        if (h > 0)
            return h + ":" + (m < 10 ? "0" : "") + m + ":" + (s < 10 ? "0" : "") + s
        return m + ":" + (s < 10 ? "0" : "") + s
    }

    Component.onCompleted: {
        if (magnetLink) backend.playMagnet(magnetLink, -1)
    }

    Component.onDestruction: {
        mpvPlayer.paused = true
        backend.stopStream()
    }

    Connections {
        target: backend

        function onProgressReceived(progressData) { progress = progressData }

        function onStreamPathChanged() {
            console.log("onStreamPathChanged - ready:", backend.isStreamReady, "path:", backend.streamPath)
            tryLoadStream()
        }

        function onIsStreamReadyChanged() {
            console.log("onIsStreamReadyChanged - ready:", backend.isStreamReady, "path:", backend.streamPath)
            tryLoadStream()
        }
    }

    function tryLoadStream() {
        if (backend.isStreamReady && backend.streamPath) {
            console.log("Stream ready! Loading:", backend.streamPath)
            mpvPlayer.source = backend.streamPath
            mpvPlayer.paused = false
            hideControlsTimer.start()
        }
    }
}
