import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Item {
    id: root

    property var movie: null
    property var movieDetails: null

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
            case "yts":
                return "#4caf50"
            default:
                return "#666"
        }
    }

    Rectangle {
        anchors.fill: parent
        color: "#0a0908"

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            // Header tipo breadcrumb
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 52
                color: "#0d0c0b"

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 28
                    anchors.rightMargin: 28
                    spacing: 8

                    Text {
                        text: "Películas"
                        font.pixelSize: 13
                        color: "#8a837c"

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.backRequested()
                        }
                    }

                    Text {
                        text: "/"
                        font.pixelSize: 13
                        color: "#2a2521"
                    }

                    Text {
                        text: movie ? movie.title : ""
                        font.pixelSize: 13
                        color: "#f5f3f0"
                        elide: Text.ElideRight
                        Layout.fillWidth: true
                    }

                    Item { Layout.fillWidth: true }

                    // Botón volver
                    Rectangle {
                        width: 80
                        height: 30
                        color: "transparent"
                        border.color: "#2a2521"
                        border.width: 1
                        radius: 6

                        Text {
                            anchors.centerIn: parent
                            text: "← Volver"
                            font.pixelSize: 12
                            color: "#8a837c"
                        }

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.backRequested()
                            hoverEnabled: true
                            onEntered: parent.border.color = "#ff6b35"
                            onExited: parent.border.color = "#2a2521"
                        }
                    }
                }

                Rectangle {
                    anchors.bottom: parent.bottom
                    width: parent.width
                    height: 1
                    color: "#1a1613"
                }
            }

            // Content area with scroll
            ScrollView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true

                Item {
                    width: parent.width
                    implicitHeight: contentColumn.implicitHeight + 48

                    ColumnLayout {
                        id: contentColumn
                        width: Math.min(parent.width - 56, 1200)
                        anchors.horizontalCenter: parent.horizontalCenter
                        anchors.top: parent.top
                        anchors.topMargin: 32
                        spacing: 32

                        // Movie header (poster + info)
                        RowLayout {
                            Layout.fillWidth: true
                            spacing: 32

                            // Poster
                            Rectangle {
                                Layout.preferredWidth: 240
                                Layout.preferredHeight: 360
                                Layout.alignment: Qt.AlignTop
                                color: "#1a1613"
                                radius: 10
                                clip: true

                                Image {
                                    anchors.fill: parent
                                    source: movie ? movie.posterUrl : ""
                                    fillMode: Image.PreserveAspectCrop
                                    asynchronous: true
                                    cache: true
                                    smooth: true

                                    Rectangle {
                                        anchors.fill: parent
                                        color: "#1a1613"
                                        visible: parent.status !== Image.Ready
                                        z: -1

                                        BusyIndicator {
                                            anchors.centerIn: parent
                                            running: parent.visible && parent.parent.status === Image.Loading
                                        }
                                    }
                                }
                            }

                            // Info
                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.alignment: Qt.AlignTop
                                spacing: 0

                                // Géneros como chips pequeños
                                Row {
                                    spacing: 6
                                    Layout.bottomMargin: 14

                                    Repeater {
                                        model: movie ? movie.genres.split(",").map(function(g) { return g.trim() }).slice(0, 4) : []

                                        delegate: Rectangle {
                                            width: genreText.width + 14
                                            height: 22
                                            color: "#171513"
                                            border.color: "#2a2521"
                                            border.width: 1
                                            radius: 4

                                            Text {
                                                id: genreText
                                                anchors.centerIn: parent
                                                text: modelData
                                                font.pixelSize: 11
                                                color: "#8a837c"
                                            }
                                        }
                                    }
                                }

                                Text {
                                    text: movie ? movie.title : ""
                                    font.pixelSize: 38
                                    font.weight: Font.Bold
                                    color: "#f5f3f0"
                                    wrapMode: Text.WordWrap
                                    Layout.fillWidth: true
                                    font.letterSpacing: -1
                                }

                                RowLayout {
                                    spacing: 16
                                    Layout.topMargin: 10
                                    Layout.bottomMargin: 16

                                    Text {
                                        text: movie ? movie.year : ""
                                        font.pixelSize: 15
                                        color: "#8a837c"
                                    }

                                    Rectangle {
                                        width: 1
                                        height: 14
                                        color: "#2a2521"
                                    }

                                    // Rating
                                    RowLayout {
                                        spacing: 4
                                        visible: movie && movie.rating > 0

                                        Text {
                                            text: "★"
                                            font.pixelSize: 14
                                            color: "#ff6b35"
                                        }

                                        Text {
                                            text: movie ? movie.rating.toFixed(1) : ""
                                            font.pixelSize: 14
                                            font.weight: Font.SemiBold
                                            color: "#ff6b35"
                                        }
                                    }

                                    Rectangle {
                                        width: 1
                                        height: 14
                                        color: "#2a2521"
                                        visible: movieDetails && movieDetails.runtime
                                    }

                                    Text {
                                        text: movieDetails ? movieDetails.runtime + " min" : ""
                                        font.pixelSize: 15
                                        color: "#8a837c"
                                        visible: movieDetails && movieDetails.runtime
                                    }
                                }

                                Text {
                                    text: movieDetails ? movieDetails.description : "Cargando detalles..."
                                    font.pixelSize: 14
                                    color: "#b0a99f"
                                    wrapMode: Text.WordWrap
                                    Layout.fillWidth: true
                                    lineHeight: 1.55
                                }
                            }
                        }

                        // Torrents section
                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 10
                            visible: movieDetails && movieDetails.torrents && movieDetails.torrents.length > 0

                            Text {
                                text: "Torrents disponibles"
                                font.pixelSize: 18
                                font.weight: Font.SemiBold
                                color: "#f5f3f0"
                                Layout.bottomMargin: 4
                            }

                            Repeater {
                                model: movieDetails ? movieDetails.torrents : []

                                delegate: Rectangle {
                                    Layout.fillWidth: true
                                    height: 90
                                    color: torrentMouseArea.containsMouse ? "#141210" : (index === 0 ? "#100e0c" : "#0d0c0b")
                                    radius: 8
                                    border.color: index === 0 ? "#ff6b3566" : (torrentMouseArea.containsMouse ? "#2a2521" : "#1a1613")
                                    border.width: index === 0 ? 1 : 1

                                    MouseArea {
                                        id: torrentMouseArea
                                        anchors.fill: parent
                                        hoverEnabled: true
                                        cursorShape: Qt.PointingHandCursor
                                        onClicked: root.playRequested(modelData.magnetLink)
                                    }

                                    RowLayout {
                                        anchors.fill: parent
                                        anchors.leftMargin: 18
                                        anchors.rightMargin: 18
                                        spacing: 16

                                        // Indicador de posición (primero = destacado)
                                        Rectangle {
                                            width: 3
                                            height: 48
                                            radius: 2
                                            color: index === 0 ? "#ff6b35" : "#2a2521"
                                            Layout.alignment: Qt.AlignVCenter
                                        }

                                        ColumnLayout {
                                            Layout.fillWidth: true
                                            spacing: 8

                                            Text {
                                                text: modelData.fileName || modelData.title || "Unknown"
                                                font.pixelSize: 13
                                                font.weight: Font.Medium
                                                color: "#f5f3f0"
                                                wrapMode: Text.NoWrap
                                                elide: Text.ElideRight
                                                Layout.fillWidth: true
                                            }

                                            RowLayout {
                                                spacing: 10

                                                // Quality badge
                                                Rectangle {
                                                    width: qualityText.width + 16
                                                    height: 24
                                                    color: index === 0 ? "#ff6b35" : "#1a1613"
                                                    border.color: index === 0 ? "#ff6b35" : "#2a2521"
                                                    border.width: 1
                                                    radius: 4

                                                    Text {
                                                        id: qualityText
                                                        anchors.centerIn: parent
                                                        text: modelData.quality
                                                        font.pixelSize: 11
                                                        font.weight: Font.Bold
                                                        color: index === 0 ? "#0a0908" : "#f5f3f0"
                                                    }
                                                }

                                                Text {
                                                    text: modelData.type ? modelData.type.toUpperCase() : ""
                                                    font.pixelSize: 11
                                                    color: "#8a837c"
                                                }

                                                Text {
                                                    text: modelData.size
                                                    font.pixelSize: 11
                                                    color: "#8a837c"
                                                }

                                                // Provider badge
                                                Rectangle {
                                                    width: providerText.width + 12
                                                    height: 20
                                                    color: getProviderColor(modelData.provider)
                                                    radius: 3
                                                    visible: modelData.provider !== undefined

                                                    Text {
                                                        id: providerText
                                                        anchors.centerIn: parent
                                                        text: modelData.provider ? modelData.provider.toUpperCase() : ""
                                                        font.pixelSize: 10
                                                        font.weight: Font.Bold
                                                        color: "#fff"
                                                    }
                                                }

                                                // Seeds
                                                RowLayout {
                                                    spacing: 4
                                                    visible: modelData.seeds > 0

                                                    Rectangle {
                                                        width: 6; height: 6; radius: 3
                                                        color: "#4caf50"
                                                    }

                                                    Text {
                                                        text: modelData.seeds + " seeds"
                                                        font.pixelSize: 11
                                                        color: "#8a837c"
                                                    }
                                                }
                                            }

                                            // Subtitles
                                            Flow {
                                                Layout.fillWidth: true
                                                spacing: 6
                                                visible: modelData.subtitles && modelData.subtitles.length > 0

                                                Repeater {
                                                    model: modelData.subtitles || []
                                                    delegate: Rectangle {
                                                        width: subtitleText.width + 10
                                                        height: 18
                                                        color: "#171513"
                                                        radius: 3
                                                        border.color: "#2a2521"
                                                        border.width: 1

                                                        Text {
                                                            id: subtitleText
                                                            anchors.centerIn: parent
                                                            text: modelData
                                                            font.pixelSize: 10
                                                            color: "#8a837c"
                                                        }
                                                    }
                                                }
                                            }
                                        }

                                        // Play button
                                        Rectangle {
                                            width: 110
                                            height: 38
                                            color: index === 0 ? "#ff6b35" : "transparent"
                                            border.color: index === 0 ? "#ff6b35" : "#2a2521"
                                            border.width: 1
                                            radius: 6
                                            Layout.alignment: Qt.AlignVCenter

                                            Text {
                                                anchors.centerIn: parent
                                                text: "▶ Reproducir"
                                                font.pixelSize: 13
                                                font.weight: Font.SemiBold
                                                color: index === 0 ? "#0a0908" : "#f5f3f0"
                                            }
                                        }
                                    }
                                }
                            }
                        }

                        // Loading indicator
                        BusyIndicator {
                            Layout.alignment: Qt.AlignHCenter
                            running: backend.isLoading && !movieDetails
                            visible: running
                        }
                    }
                }
            }
        }
    }

    Component.onCompleted: {
        if (movie) {
            backend.getMovieDetails(movie)
        }
    }

    Connections {
        target: backend
        function onMovieDetailsReceived(details) {
            movieDetails = details
        }
    }
}
