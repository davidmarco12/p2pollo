import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Item {
    id: root

    property var series: null
    property var seriesDetails: null
    property int selectedSeason: 0

    signal playRequested(string magnetLink)
    signal backRequested()

    function getProviderColor(provider) {
        if (!provider) return "#666"
        switch(provider.toLowerCase()) {
            case "rargb":
            case "rargbto":
                return "#00a8e1"
            case "thepiratebay":
            case "tpb":
                return "#6a1b9a"
            default:
                return "#666"
        }
    }

    function currentSeasonEpisodes() {
        if (!seriesDetails || !seriesDetails.seasons || seriesDetails.seasons.length === 0)
            return []
        return seriesDetails.seasons[selectedSeason].episodes || []
    }

    Rectangle {
        anchors.fill: parent
        color: "#0a0908"

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            // Header
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 60
                color: "#1a1613"

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 16
                    anchors.rightMargin: 16
                    spacing: 12

                    Button {
                        text: "← Volver"
                        font.pixelSize: 14
                        onClicked: root.backRequested()

                        background: Rectangle {
                            color: parent.hovered ? "#2a2521" : "transparent"
                            radius: 10
                        }

                        contentItem: Text {
                            text: parent.text
                            color: "#ff6b35"
                            font: parent.font
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter
                        }
                    }

                    Text {
                        text: series ? series.title : ""
                        font.pixelSize: 18
                        font.weight: Font.Medium
                        color: "#f5f3f0"
                        elide: Text.ElideRight
                        Layout.fillWidth: true
                    }
                }
            }

            // Content
            ScrollView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true

                Item {
                    width: parent.width
                    implicitHeight: contentColumn.implicitHeight + 48

                    ColumnLayout {
                        id: contentColumn
                        width: Math.min(parent.width - 48, 1200)
                        anchors.horizontalCenter: parent.horizontalCenter
                        anchors.top: parent.top
                        anchors.topMargin: 24
                        spacing: 24

                        // Serie info (poster + datos)
                        RowLayout {
                            Layout.fillWidth: true
                            spacing: 24

                            // Poster
                            Image {
                                Layout.preferredWidth: 200
                                Layout.preferredHeight: 300
                                Layout.alignment: Qt.AlignTop
                                source: series ? series.posterUrl : ""
                                fillMode: Image.PreserveAspectFit
                                asynchronous: true
                                cache: true
                                smooth: true

                                Rectangle {
                                    anchors.fill: parent
                                    color: "#1a1613"
                                    visible: parent.status !== Image.Ready
                                    z: -1
                                }
                            }

                            // Info
                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.alignment: Qt.AlignTop
                                spacing: 10

                                Text {
                                    text: series ? series.title : ""
                                    font.pixelSize: 28
                                    font.weight: Font.Bold
                                    color: "#f5f3f0"
                                    wrapMode: Text.WordWrap
                                    Layout.fillWidth: true
                                }

                                RowLayout {
                                    spacing: 12

                                    Text {
                                        text: series ? series.year : ""
                                        font.pixelSize: 15
                                        color: "#8a837c"
                                    }

                                    Rectangle {
                                        width: 60
                                        height: 26
                                        color: "#ff6b3533"
                                        border.color: "#ff6b35"
                                        border.width: 1
                                        radius: 4
                                        visible: series && series.rating > 0

                                        Text {
                                            anchors.centerIn: parent
                                            text: series ? "★ " + series.rating.toFixed(1) : ""
                                            font.pixelSize: 13
                                            font.weight: Font.Bold
                                            color: "#ff6b35"
                                        }
                                    }
                                }

                                Text {
                                    text: series ? series.genres : ""
                                    font.pixelSize: 13
                                    color: "#ff6b35"
                                }

                                Text {
                                    text: seriesDetails ? seriesDetails.description : "Cargando detalles..."
                                    font.pixelSize: 14
                                    color: "#ccc"
                                    wrapMode: Text.WordWrap
                                    Layout.fillWidth: true
                                    Layout.topMargin: 8
                                }
                            }
                        }

                        // Temporadas + episodios
                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 0
                            visible: seriesDetails && seriesDetails.seasons && seriesDetails.seasons.length > 0

                            Text {
                                text: "Episodios"
                                font.pixelSize: 22
                                font.weight: Font.Bold
                                color: "#f5f3f0"
                                Layout.bottomMargin: 12
                            }

                            // Tab bar de temporadas
                            ScrollView {
                                Layout.fillWidth: true
                                Layout.preferredHeight: 44
                                ScrollBar.vertical.policy: ScrollBar.AlwaysOff
                                clip: true

                                Row {
                                    spacing: 8

                                    Repeater {
                                        model: seriesDetails ? seriesDetails.seasons : []

                                        delegate: Rectangle {
                                            width: seasonLabel.width + 24
                                            height: 36
                                            color: selectedSeason === index ? "#ff6b35" : "#1a1613"
                                            border.color: selectedSeason === index ? "#ff6b35" : "#2a2521"
                                            border.width: 1
                                            radius: 6

                                            Text {
                                                id: seasonLabel
                                                anchors.centerIn: parent
                                                text: "Temporada " + modelData.number
                                                font.pixelSize: 13
                                                font.weight: selectedSeason === index ? Font.Bold : Font.Normal
                                                color: selectedSeason === index ? "#fff" : "#ccc"
                                            }

                                            MouseArea {
                                                anchors.fill: parent
                                                cursorShape: Qt.PointingHandCursor
                                                onClicked: selectedSeason = index
                                            }
                                        }
                                    }
                                }
                            }

                            // Lista de episodios
                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.topMargin: 12
                                spacing: 6

                                Repeater {
                                    model: root.currentSeasonEpisodes()

                                    delegate: Rectangle {
                                        Layout.fillWidth: true
                                        height: 56
                                        color: epMouseArea.containsMouse ? "#1a2030" : "#0f1520"
                                        radius: 6
                                        border.color: "#2a2521"
                                        border.width: 1

                                        MouseArea {
                                            id: epMouseArea
                                            anchors.fill: parent
                                            hoverEnabled: true
                                            cursorShape: Qt.PointingHandCursor
                                            onClicked: {
                                                // Limpiar antes de abrir para que una respuesta stale
                                                // de un click anterior no pise los datos del episodio nuevo.
                                                torrentDialog.torrents = []
                                                torrentDialog.episodeTitle = modelData.title || ""
                                                torrentDialog.season = modelData.season
                                                torrentDialog.episode = modelData.episode
                                                torrentDialog.open()
                                                backend.getEpisodeTorrents(series.title, series.imdbId || "", modelData.season, modelData.episode)
                                            }
                                        }

                                        RowLayout {
                                            anchors.fill: parent
                                            anchors.leftMargin: 16
                                            anchors.rightMargin: 16
                                            spacing: 12

                                            // Numero de episodio
                                            Rectangle {
                                                width: 36
                                                height: 36
                                                color: "#1a1613"
                                                radius: 4

                                                Text {
                                                    anchors.centerIn: parent
                                                    text: "E" + String(modelData.episode).padStart(2, "0")
                                                    font.pixelSize: 12
                                                    font.weight: Font.Bold
                                                    color: "#ff6b35"
                                                }
                                            }

                                            // Titulo del episodio
                                            ColumnLayout {
                                                Layout.fillWidth: true
                                                spacing: 2

                                                Text {
                                                    text: modelData.title || "Episodio " + modelData.episode
                                                    font.pixelSize: 14
                                                    font.weight: Font.Medium
                                                    color: "#f5f3f0"
                                                    elide: Text.ElideRight
                                                    Layout.fillWidth: true
                                                }

                                                Text {
                                                    text: modelData.airDate || ""
                                                    font.pixelSize: 11
                                                    color: "#8a837c"
                                                    visible: modelData.airDate !== ""
                                                }
                                            }

                                            // Play hint
                                            Text {
                                                text: "▶"
                                                font.pixelSize: 18
                                                color: epMouseArea.containsMouse ? "#ff6b35" : "#2a2521"
                                            }
                                        }
                                    }
                                }
                            }
                        }

                        // Loading
                        BusyIndicator {
                            Layout.alignment: Qt.AlignHCenter
                            running: backend.isLoading && !seriesDetails
                            visible: running
                        }

                        // Mensaje sin episodios
                        Text {
                            text: "No se encontraron episodios para esta serie."
                            font.pixelSize: 14
                            color: "#8a837c"
                            Layout.alignment: Qt.AlignHCenter
                            visible: seriesDetails && (!seriesDetails.seasons || seriesDetails.seasons.length === 0)
                        }
                    }
                }
            }
        }
    }

    // Dialog de torrents del episodio
    Dialog {
        id: torrentDialog
        modal: true
        width: Math.min(parent.width - 48, 900)
        height: Math.min(parent.height - 80, 600)
        anchors.centerIn: parent

        property string episodeTitle: ""
        property int season: 0
        property int episode: 0
        property var torrents: []

        title: series ? (series.title + " S" + String(season).padStart(2,"0") + "E" + String(episode).padStart(2,"0")) : ""

        background: Rectangle {
            color: "#0f1520"
            radius: 10
            border.color: "#ff6b3533"
            border.width: 1
        }

        header: Rectangle {
            color: "#1a1613"
            height: 56
            radius: 10

            Text {
                anchors.centerIn: parent
                text: torrentDialog.title
                font.pixelSize: 16
                font.weight: Font.Bold
                color: "#f5f3f0"
            }

            Button {
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                anchors.rightMargin: 12
                text: "✕"
                font.pixelSize: 16
                onClicked: torrentDialog.close()

                background: Rectangle { color: "transparent" }
                contentItem: Text {
                    text: parent.text
                    color: "#8a837c"
                    font: parent.font
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
            }
        }

        contentItem: Item {
            ColumnLayout {
                anchors.fill: parent
                spacing: 8

                // Buscando...
                BusyIndicator {
                    Layout.alignment: Qt.AlignHCenter
                    running: backend.isLoading && torrentDialog.torrents.length === 0
                    visible: running
                }

                Text {
                    text: "No se encontraron torrents para este episodio."
                    font.pixelSize: 14
                    color: "#8a837c"
                    Layout.alignment: Qt.AlignHCenter
                    visible: !backend.isLoading && torrentDialog.torrents.length === 0
                }

                // Lista de torrents
                ScrollView {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    clip: true
                    visible: torrentDialog.torrents.length > 0

                    ColumnLayout {
                        width: parent.width
                        height: implicitHeight
                        spacing: 8

                        Repeater {
                            model: torrentDialog.torrents

                            delegate: Rectangle {
                                Layout.fillWidth: true
                                height: 90
                                color: tMouseArea.containsMouse ? "#1a2030" : "#0a0f1a"
                                radius: 6
                                border.color: "#2a2521"
                                border.width: 1

                                MouseArea {
                                    id: tMouseArea
                                    anchors.fill: parent
                                    hoverEnabled: true
                                    onClicked: {
                                        root.playRequested(modelData.magnetLink)
                                        torrentDialog.close()
                                    }
                                }

                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: 12
                                    spacing: 12

                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        spacing: 6

                                        Text {
                                            text: modelData.fileName || modelData.title || "Unknown"
                                            font.pixelSize: 13
                                            color: "#f5f3f0"
                                            elide: Text.ElideRight
                                            Layout.fillWidth: true
                                        }

                                        RowLayout {
                                            spacing: 8

                                            Rectangle {
                                                width: 70
                                                height: 24
                                                color: "#ff6b35"
                                                radius: 4

                                                Text {
                                                    anchors.centerIn: parent
                                                    text: modelData.quality
                                                    font.pixelSize: 11
                                                    font.weight: Font.Bold
                                                    color: "#fff"
                                                }
                                            }

                                            Text {
                                                text: modelData.size
                                                font.pixelSize: 12
                                                color: "#8a837c"
                                            }

                                            Rectangle {
                                                width: provText.width + 10
                                                height: 20
                                                color: root.getProviderColor(modelData.provider)
                                                radius: 3
                                                visible: modelData.provider !== undefined

                                                Text {
                                                    id: provText
                                                    anchors.centerIn: parent
                                                    text: modelData.provider ? modelData.provider.toUpperCase() : ""
                                                    font.pixelSize: 10
                                                    font.weight: Font.Bold
                                                    color: "#fff"
                                                }
                                            }
                                        }
                                    }

                                    ColumnLayout {
                                        Layout.alignment: Qt.AlignTop
                                        spacing: 6

                                        Rectangle {
                                            width: 90
                                            height: 34
                                            color: "#ff6b35"
                                            radius: 5

                                            Text {
                                                anchors.centerIn: parent
                                                text: "▶ Reproducir"
                                                font.pixelSize: 13
                                                font.weight: Font.Bold
                                                color: "#fff"
                                            }
                                        }

                                        Text {
                                            text: "🌱 " + modelData.seeds
                                            font.pixelSize: 11
                                            color: "#aaa"
                                            Layout.alignment: Qt.AlignHCenter
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        onOpened: {
            torrentDialog.torrents = []
        }

        onClosed: {
            torrentDialog.torrents = []
        }
    }

    Component.onCompleted: {
        if (series) {
            backend.getSeriesDetails(series)
        }
    }

    Connections {
        target: backend
        function onSeriesDetailsReceived(details) {
            seriesDetails = details
            selectedSeason = 0
        }
        function onEpisodeTorrentsReceived(torrents) {
            torrentDialog.torrents = torrents
        }
    }
}
