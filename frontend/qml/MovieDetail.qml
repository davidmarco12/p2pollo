import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Item {
    id: root

    property var movie: null
    property var movieDetails: null

    signal playRequested(string magnetLink)
    signal backRequested()

    // Helper function para obtener color por provider
    function getProviderColor(provider) {
        if (!provider) return "#666"

        switch(provider.toLowerCase()) {
            case "rargb":
            case "rargbto":
                return "#00a8e1"  // Azul
            case "thepiratebay":
            case "tpb":
                return "#6a1b9a"  // Púrpura
            case "yts":
                return "#4caf50"  // Verde
            default:
                return "#666"     // Gris
        }
    }

    Rectangle {
        anchors.fill: parent
        color: "#0a0908"

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            // Header with back button
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 60
                color: "#1a1613"

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 16
                    anchors.rightMargin: 16
                    spacing: 12

                    // Back button
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

                    Item { Layout.fillWidth: true }
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
                        width: Math.min(parent.width - 48, 1200)
                        anchors.horizontalCenter: parent.horizontalCenter
                        anchors.top: parent.top
                        anchors.margins: 24
                        spacing: 24

                    // Movie header (poster + info)
                    RowLayout {
                        Layout.fillWidth: true
                        spacing: 24

                        // Poster
                        Image {
                            Layout.preferredWidth: 300
                            Layout.preferredHeight: 450
                            Layout.alignment: Qt.AlignTop
                            source: movie ? movie.posterUrl : ""
                            fillMode: Image.PreserveAspectFit
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

                        // Info
                        ColumnLayout {
                            Layout.fillWidth: true
                            Layout.alignment: Qt.AlignTop
                            spacing: 12

                            Text {
                                text: movie ? movie.title : ""
                                font.pixelSize: 32
                                font.weight: Font.Bold
                                color: "#f5f3f0"
                                wrapMode: Text.WordWrap
                                Layout.fillWidth: true
                            }

                            RowLayout {
                                spacing: 16

                                // Year
                                Text {
                                    text: movie ? movie.year : ""
                                    font.pixelSize: 16
                                    color: "#8a837c"
                                }

                                // Rating
                                Rectangle {
                                    width: 60
                                    height: 30
                                    color: "#ff6b3533"
                                    border.color: "#ff6b35"
                                    border.width: 1
                                    radius: 4

                                    Text {
                                        anchors.centerIn: parent
                                        text: movie ? "★ " + movie.rating.toFixed(1) : ""
                                        font.pixelSize: 14
                                        font.weight: Font.Bold
                                        color: "#ff6b35"
                                    }
                                }

                                // Runtime
                                Text {
                                    text: movieDetails ? movieDetails.runtime + " min" : ""
                                    font.pixelSize: 16
                                    color: "#8a837c"
                                    visible: movieDetails && movieDetails.runtime
                                }
                            }

                            // Genres
                            Text {
                                text: movie ? movie.genres : ""
                                font.pixelSize: 14
                                color: "#ff6b35"
                                Layout.topMargin: 8
                            }

                            // Description
                            Text {
                                text: movieDetails ? movieDetails.description : "Cargando detalles..."
                                font.pixelSize: 14
                                color: "#ccc"
                                wrapMode: Text.WordWrap
                                Layout.fillWidth: true
                                Layout.topMargin: 16
                            }
                        }
                    }

                    // Torrents section
                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 12
                        visible: movieDetails && movieDetails.torrents && movieDetails.torrents.length > 0

                        Text {
                            text: "Descargar"
                            font.pixelSize: 24
                            font.weight: Font.Bold
                            color: "#f5f3f0"
                        }

                        // Torrent list
                        Repeater {
                            model: movieDetails ? movieDetails.torrents : []

                            delegate: Rectangle {
                                Layout.fillWidth: true
                                height: 110
                                color: torrentMouseArea.containsMouse ? "#2a3f54" : "#0f1923"
                                radius: 8
                                border.color: "#444"
                                border.width: 1

                                MouseArea {
                                    id: torrentMouseArea
                                    anchors.fill: parent
                                    hoverEnabled: true
                                    onClicked: {
                                        root.playRequested(modelData.magnetLink)
                                    }
                                }

                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: 16
                                    spacing: 16

                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        spacing: 8

                                        // Filename title
                                        Text {
                                            text: modelData.fileName || modelData.title || "Unknown"
                                            font.pixelSize: 14
                                            font.weight: Font.Medium
                                            color: "#f5f3f0"
                                            wrapMode: Text.NoWrap
                                            elide: Text.ElideRight
                                            Layout.fillWidth: true
                                        }

                                        RowLayout {
                                            spacing: 12

                                            // Quality badge
                                            Rectangle {
                                                width: 80
                                                height: 28
                                                color: "#ff6b35"
                                                radius: 4

                                                Text {
                                                    anchors.centerIn: parent
                                                    text: modelData.quality
                                                    font.pixelSize: 12
                                                    font.weight: Font.Bold
                                                    color: "#fff"
                                                }
                                            }

                                            // Type
                                            Text {
                                                text: modelData.type.toUpperCase()
                                                font.pixelSize: 12
                                                color: "#8a837c"
                                            }

                                            // Size
                                            Text {
                                                text: modelData.size
                                                font.pixelSize: 12
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
                                        }

                                        // Subtitles row
                                        Flow {
                                            Layout.fillWidth: true
                                            spacing: 6
                                            visible: modelData.subtitles && modelData.subtitles.length > 0

                                            Repeater {
                                                model: modelData.subtitles || []
                                                delegate: Rectangle {
                                                    width: subtitleText.width + 12
                                                    height: 20
                                                    color: "#1a1613"
                                                    radius: 3
                                                    border.color: "#444"
                                                    border.width: 1

                                                    Text {
                                                        id: subtitleText
                                                        anchors.centerIn: parent
                                                        text: modelData
                                                        font.pixelSize: 10
                                                        color: "#aaa"
                                                    }
                                                }
                                            }
                                        }
                                    }

                                    // Right column: Play button + Seeds/Peers
                                    ColumnLayout {
                                        Layout.alignment: Qt.AlignTop
                                        spacing: 8

                                        // Play button
                                        Rectangle {
                                            width: 100
                                            height: 40
                                            color: "#ff6b35"
                                            radius: 6

                                            Text {
                                                anchors.centerIn: parent
                                                text: "▶ Reproducir"
                                                font.pixelSize: 14
                                                font.weight: Font.Bold
                                                color: "#fff"
                                            }
                                        }

                                        // Seeds and peers
                                        Text {
                                            text: "🌱 " + modelData.seeds + " seeds\n👥 " + modelData.peers + " peers"
                                            font.pixelSize: 11
                                            color: "#aaa"
                                            horizontalAlignment: Text.AlignHCenter
                                            Layout.alignment: Qt.AlignHCenter
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

    // Load movie details when component is created
    Component.onCompleted: {
        if (movie) {
            backend.getMovieDetails(movie)
        }
    }

    // Handle movie details response
    Connections {
        target: backend
        function onMovieDetailsReceived(details) {
            movieDetails = details
        }
    }
}
