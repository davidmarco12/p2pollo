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

    // Timer para ocultar controles automáticamente
    Timer {
        id: hideControlsTimer
        interval: 3000  // 3 segundos sin movimiento
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

        // MouseArea para detectar movimiento del cursor
        MouseArea {
            anchors.fill: parent
            hoverEnabled: true
            propagateComposedEvents: true
            acceptedButtons: Qt.NoButton  // No capturar clicks, solo movimiento
            cursorShape: controlsVisible ? Qt.ArrowCursor : Qt.BlankCursor

            onPositionChanged: {
                // Mostrar controles al mover el mouse
                if (!controlsVisible) {
                    controlsVisible = true
                }
                // Resetear timer
                hideControlsTimer.restart()
            }

            onEntered: {
                controlsVisible = true
                hideControlsTimer.restart()
            }
        }

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            // Video area
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: "#000"

                // Loading overlay
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

                        // Progress info
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

                // Overlay de buffering mid-playback: aparece cuando mpv se tilda esperando datos
                // (el usuario adelantó más allá del buffer descargado)
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

                // MPV Player (embedded)
                // Note: Always visible so Qt creates the OpenGL renderer immediately
                // The loading overlay (z: 10) covers this when not ready
                MpvObject {
                    id: mpvPlayer
                    anchors.fill: parent

                    Component.onCompleted: {
                        console.log("MpvObject created")
                    }

                    onPausedChanged: {
                        console.log("Paused:", paused)
                    }

                    onPositionChanged: {
                        // Update seek slider without triggering onMoved
                        if (!seekSlider.pressed) {
                            seekSlider.value = position
                        }
                    }

                    onVolumeChanged: {
                        // Update volume slider without triggering onMoved
                        if (!volumeSlider.pressed) {
                            volumeSlider.value = volume
                        }
                    }

                    onDurationChanged: {
                        console.log("Duration:", duration)
                    }
                }
            }

            // Controls bar
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: controlsVisible ? 80 : 0
                color: "#1a1613"
                visible: backend.isStreamReady && controlsVisible
                opacity: controlsVisible ? 1.0 : 0.0

                Behavior on Layout.preferredHeight {
                    NumberAnimation { duration: 200; easing.type: Easing.InOutQuad }
                }
                Behavior on opacity {
                    NumberAnimation { duration: 200 }
                }

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 12
                    spacing: 8

                    // Progress bar
                    Slider {
                        id: seekSlider
                        Layout.fillWidth: true
                        from: 0
                        to: mpvPlayer.duration > 0 ? mpvPlayer.duration : 1
                        value: 0
                        enabled: mpvPlayer.duration > 0

                        onMoved: {
                            mpvPlayer.setPosition(value)
                        }

                        background: Rectangle {
                            x: seekSlider.leftPadding
                            y: seekSlider.topPadding + seekSlider.availableHeight / 2 - height / 2
                            width: seekSlider.availableWidth
                            height: 4
                            radius: 2
                            color: "#2a2521"

                            // Barra gris: porción del video descargada (buffer disponible)
                            Rectangle {
                                width: {
                                    if (!progress || progress.totalSize <= 0) return 0
                                    var frac = Math.min(progress.headWritten / progress.totalSize, 1.0)
                                    return frac * parent.width
                                }
                                height: parent.height
                                color: "#5a5551"
                                radius: 2
                            }

                            // Barra naranja: posición actual de reproducción
                            Rectangle {
                                width: seekSlider.visualPosition * parent.width
                                height: parent.height
                                color: "#ff6b35"
                                radius: 2
                            }
                        }

                        handle: Rectangle {
                            x: seekSlider.leftPadding + seekSlider.visualPosition * (seekSlider.availableWidth - width)
                            y: seekSlider.topPadding + seekSlider.availableHeight / 2 - height / 2
                            width: 16
                            height: 16
                            radius: 8
                            color: "#ff6b35"
                            border.color: "#f5f3f0"
                            border.width: 2
                        }
                    }

                    // Playback controls
                    RowLayout {
                        Layout.fillWidth: true
                        spacing: 12

                        // Back button
                        Button {
                            text: "←"
                            font.pixelSize: 20
                            onClicked: root.backRequested()

                            background: Rectangle {
                                color: parent.hovered ? "#2a3f54" : "transparent"
                                radius: 4
                            }

                            contentItem: Text {
                                text: parent.text
                                color: "#f5f3f0"
                                font: parent.font
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                            }
                        }

                        // Rewind button
                        Button {
                            text: "⏪"
                            font.pixelSize: 18
                            onClicked: mpvPlayer.seek(-10)

                            background: Rectangle {
                                color: parent.hovered ? "#2a3f54" : "transparent"
                                radius: 4
                            }

                            contentItem: Text {
                                text: parent.text
                                color: "#f5f3f0"
                                font: parent.font
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                            }
                        }

                        // Play/Pause button
                        Button {
                            text: mpvPlayer.paused ? "▶" : "⏸"
                            font.pixelSize: 20
                            onClicked: {
                                mpvPlayer.paused = !mpvPlayer.paused
                            }

                            background: Rectangle {
                                implicitWidth: 50
                                implicitHeight: 50
                                color: parent.hovered ? "#ff8555" : "#ff6b35"
                                radius: 25
                            }

                            contentItem: Text {
                                text: parent.text
                                color: "#0a0908"
                                font.pixelSize: parent.font.pixelSize
                                font.weight: Font.Medium
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                            }
                        }

                        // Forward button
                        Button {
                            text: "⏩"
                            font.pixelSize: 18
                            onClicked: mpvPlayer.seek(30)

                            background: Rectangle {
                                color: parent.hovered ? "#2a3f54" : "transparent"
                                radius: 4
                            }

                            contentItem: Text {
                                text: parent.text
                                color: "#f5f3f0"
                                font: parent.font
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                            }
                        }

                        Item { Layout.fillWidth: true }

                        // Time display
                        Text {
                            text: formatTime(mpvPlayer.position) + " / " + formatTime(mpvPlayer.duration)
                            font.pixelSize: 14
                            color: "#f5f3f0"
                        }

                        // Volume control
                        RowLayout {
                            spacing: 8

                            Text {
                                text: "🔊"
                                font.pixelSize: 16
                            }

                            Slider {
                                id: volumeSlider
                                from: 0
                                to: 100
                                value: 50
                                Layout.preferredWidth: 100

                                onMoved: {
                                    console.log("VolumeSlider moved to:", value)
                                    mpvPlayer.volume = Math.round(value)
                                }
                            }

                            Text {
                                text: Math.round(volumeSlider.value) + "%"
                                font.pixelSize: 12
                                color: "#8a837c"
                                Layout.preferredWidth: 40
                            }
                        }

                        // Subtitles button with menu
                        Button {
                            id: subtitlesButton
                            text: "CC"
                            font.pixelSize: 14
                            font.bold: true
                            property int currentSubTrack: -1  // -1 = no seleccionado aun, 0 = desactivado
                            onClicked: {
                                subtitlesMenu.popup()
                            }

                            background: Rectangle {
                                color: parent.hovered ? "#2a3f54" : (mpvPlayer.subtitlesEnabled ? "#ff6b35" : "transparent")
                                radius: 4
                            }

                            contentItem: Text {
                                text: parent.text
                                color: mpvPlayer.subtitlesEnabled ? "#0a0908" : "#f5f3f0"
                                font: parent.font
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                            }

                            Menu {
                                id: subtitlesMenu
                                y: -height

                                onAboutToShow: {
                                    console.log("Loading subtitle tracks...")

                                    // Remove old items (except first 2: Desactivar and separator)
                                    while (subtitlesMenu.count > 2) {
                                        subtitlesMenu.removeItem(subtitlesMenu.itemAt(2))
                                    }

                                    // Get tracks and add them
                                    var tracks = mpvPlayer.getSubtitleTracks()
                                    console.log("Found", tracks.length, "subtitle tracks")

                                    for (var i = 0; i < tracks.length; i++) {
                                        var track = tracks[i]
                                        var label = track.lang || "Unknown"
                                        if (track.title) {
                                            label += " - " + track.title
                                        }

                                        var menuItem = menuItemComponent.createObject(subtitlesMenu, {
                                            text: label,
                                            trackId: track.id
                                        })
                                        subtitlesMenu.addItem(menuItem)
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

                        // Fullscreen button
                        Button {
                            text: root.Window.window && root.Window.window.visibility === Window.FullScreen ? "🗗" : "🗖"
                            font.pixelSize: 18
                            onClicked: {
                                if (root.Window.window) {
                                    if (root.Window.window.visibility === Window.FullScreen) {
                                        root.Window.window.showNormal()
                                        controlsVisible = true
                                    } else {
                                        root.Window.window.showFullScreen()
                                        // Ocultar controles en fullscreen después de 3 seg
                                        hideControlsTimer.restart()
                                    }
                                }
                            }

                            background: Rectangle {
                                color: parent.hovered ? "#2a3f54" : "transparent"
                                radius: 4
                            }

                            contentItem: Text {
                                text: parent.text
                                color: "#f5f3f0"
                                font: parent.font
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                            }
                        }
                    }
                }
            }
        }
    }

    // Helper function to format time
    function formatTime(seconds) {
        if (!seconds || seconds < 0) return "0:00"
        var h = Math.floor(seconds / 3600)
        var m = Math.floor((seconds % 3600) / 60)
        var s = Math.floor(seconds % 60)
        if (h > 0) {
            return h + ":" + (m < 10 ? "0" : "") + m + ":" + (s < 10 ? "0" : "") + s
        }
        return m + ":" + (s < 10 ? "0" : "") + s
    }

    // Start streaming when component is created
    Component.onCompleted: {
        if (magnetLink) {
            backend.playMagnet(magnetLink, -1)
        }
    }

    // Cleanup when component is destroyed
    Component.onDestruction: {
        // Pausar mpv primero para que libmpv deje de decodificar antes de destruirse.
        // Evita crashes cuando el renderer OpenGL se libera con frames en vuelo.
        mpvPlayer.paused = true
        backend.stopStream()
    }

    // Handle progress updates
    Connections {
        target: backend

        function onProgressReceived(progressData) {
            progress = progressData
        }

        function onStreamPathChanged() {
            console.log("onStreamPathChanged called - ready:", backend.isStreamReady, "path:", backend.streamPath)
            tryLoadStream()
        }

        function onIsStreamReadyChanged() {
            console.log("onIsStreamReadyChanged called - ready:", backend.isStreamReady, "path:", backend.streamPath)
            tryLoadStream()
        }
    }

    // Helper function to load stream when both path and ready flag are set
    function tryLoadStream() {
        if (backend.isStreamReady && backend.streamPath) {
            console.log("Stream ready! Loading in mpv:", backend.streamPath)
            mpvPlayer.source = backend.streamPath
            console.log("Source set, unpausing...")
            mpvPlayer.paused = false
            console.log("Unpause command sent")
            // Iniciar timer para ocultar controles
            hideControlsTimer.start()
        } else {
            console.log("Stream not ready yet - ready:", backend.isStreamReady, "path:", backend.streamPath)
        }
    }

}
