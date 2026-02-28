import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: root
    width: 1024
    height: 768
    visible: true
    title: "p2pollo - P2P Streaming"

    color: "#0a0908"

    property var selectedMovie: null
    property var selectedSeries: null
    property bool playerActive: false

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // ── Navbar unificada ─────────────────────────────────────────────────
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 64
            visible: !root.playerActive
            color: "#0d0c0b"

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 24
                anchors.rightMargin: 24
                spacing: 0

                // Logo
                RowLayout {
                    spacing: 10

                    Image {
                        source: "qrc:/assets/images/output-estesi.png"
                        Layout.preferredWidth: 36
                        Layout.preferredHeight: 36
                        fillMode: Image.PreserveAspectFit
                        smooth: true
                    }

                    Text {
                        text: "p2pollo"
                        font.pixelSize: 19
                        font.weight: Font.Bold
                        color: "#f5f3f0"
                        font.letterSpacing: -0.5
                    }
                }

                Item { Layout.preferredWidth: 36 }

                // Tabs
                RowLayout {
                    spacing: 0

                    Repeater {
                        model: ["Películas", "Series"]

                        delegate: Item {
                            Layout.preferredWidth: 112
                            Layout.preferredHeight: 64

                            Rectangle {
                                anchors.bottom: parent.bottom
                                width: parent.width
                                height: 2
                                color: mainContent.currentIndex === index ? "#ff6b35" : "transparent"
                            }

                            Text {
                                anchors.centerIn: parent
                                text: modelData
                                font.pixelSize: 14
                                font.weight: mainContent.currentIndex === index ? Font.SemiBold : Font.Normal
                                color: mainContent.currentIndex === index ? "#f5f3f0" : "#8a837c"
                            }

                            MouseArea {
                                anchors.fill: parent
                                cursorShape: Qt.PointingHandCursor
                                onClicked: mainContent.currentIndex = index
                            }
                        }
                    }
                }

                Item { Layout.fillWidth: true }

                // Barra de búsqueda global
                Rectangle {
                    Layout.preferredWidth: 280
                    Layout.preferredHeight: 38
                    color: "#171513"
                    radius: 19
                    border.color: globalSearch.activeFocus ? "#ff6b35" : "#2a2521"
                    border.width: 1

                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: 14
                        anchors.rightMargin: 14
                        spacing: 8

                        Text {
                            text: "⌕"
                            font.pixelSize: 17
                            color: "#8a837c"
                        }

                        TextField {
                            id: globalSearch
                            Layout.fillWidth: true
                            placeholderText: mainContent.currentIndex === 0 ? "Buscar películas..." : "Buscar series..."
                            font.pixelSize: 13
                            color: "#f5f3f0"
                            placeholderTextColor: "#8a837c"

                            background: Rectangle { color: "transparent" }

                            onAccepted: {
                                var query = text.trim()
                                if (query === "") return
                                if (mainContent.currentIndex === 0) {
                                    moviesStack.push(moviesSearchPageComponent, { "searchQuery": query }, StackView.Immediate)
                                } else {
                                    seriesStack.push(seriesSearchPageComponent, { "searchQuery": query }, StackView.Immediate)
                                }
                                text = ""
                            }
                        }
                    }
                }
            }

            // Separador inferior
            Rectangle {
                anchors.bottom: parent.bottom
                width: parent.width
                height: 1
                color: "#1a1613"
            }
        }

        // ── Contenido: dos StackViews independientes, uno por tab ────────────
        StackLayout {
            id: mainContent
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: 0

            // ── Tab 0: Películas ──────────────────────────────────────────
            StackView {
                id: moviesStack

                pushEnter: Transition {
                    PropertyAnimation { property: "opacity"; from: 0; to: 1; duration: 150 }
                }
                pushExit: Transition {
                    PropertyAnimation { property: "opacity"; from: 1; to: 0; duration: 150 }
                }
                popEnter: Transition {
                    PropertyAnimation { property: "opacity"; from: 0; to: 1; duration: 150 }
                }
                popExit: Transition {
                    PropertyAnimation { property: "opacity"; from: 1; to: 0; duration: 150 }
                }

                Component.onCompleted: {
                    moviesStack.push(homePageComponent)
                }
            }

            // ── Tab 1: Series ─────────────────────────────────────────────
            StackView {
                id: seriesStack

                pushEnter: Transition {
                    PropertyAnimation { property: "opacity"; from: 0; to: 1; duration: 150 }
                }
                pushExit: Transition {
                    PropertyAnimation { property: "opacity"; from: 1; to: 0; duration: 150 }
                }
                popEnter: Transition {
                    PropertyAnimation { property: "opacity"; from: 0; to: 1; duration: 150 }
                }
                popExit: Transition {
                    PropertyAnimation { property: "opacity"; from: 1; to: 0; duration: 150 }
                }

                Component.onCompleted: {
                    seriesStack.push(seriesPageComponent)
                }
            }
        }
    }

    // ── Componentes de Películas ──────────────────────────────────────────

    Component {
        id: homePageComponent
        HomePage {
            onMovieSelected: function(movie) {
                selectedMovie = movie
                moviesStack.push(movieDetailComponent, StackView.Immediate)
            }
        }
    }

    Component {
        id: moviesSearchPageComponent
        SearchPage {
            onMovieSelected: function(movie) {
                selectedMovie = movie
                moviesStack.push(movieDetailComponent, StackView.Immediate)
            }
            onBackRequested: {
                moviesStack.pop(StackView.Immediate)
            }
        }
    }

    Component {
        id: movieDetailComponent
        MovieDetail {
            movie: selectedMovie
            onPlayRequested: function(magnetLink) {
                moviesStack.push(moviesPlayerComponent, { "magnetLink": magnetLink }, StackView.Immediate)
            }
            onBackRequested: {
                moviesStack.pop(StackView.Immediate)
            }
        }
    }

    Component {
        id: moviesPlayerComponent
        PlayerPage {
            Component.onCompleted: root.playerActive = true
            Component.onDestruction: root.playerActive = false
            onBackRequested: {
                moviesStack.pop(StackView.Immediate)
            }
        }
    }

    // ── Componentes de Series ─────────────────────────────────────────────

    Component {
        id: seriesPageComponent
        SeriesPage {
            onSeriesSelected: function(series) {
                selectedSeries = series
                seriesStack.push(seriesDetailComponent, StackView.Immediate)
            }
        }
    }

    Component {
        id: seriesSearchPageComponent
        SeriesSearchPage {
            onSeriesSelected: function(series) {
                selectedSeries = series
                seriesStack.push(seriesDetailComponent, StackView.Immediate)
            }
            onBackRequested: {
                seriesStack.pop(StackView.Immediate)
            }
        }
    }

    Component {
        id: seriesDetailComponent
        SeriesDetail {
            series: selectedSeries
            onPlayRequested: function(magnetLink) {
                seriesStack.push(seriesPlayerComponent, { "magnetLink": magnetLink }, StackView.Immediate)
            }
            onBackRequested: {
                seriesStack.pop(StackView.Immediate)
            }
        }
    }

    Component {
        id: seriesPlayerComponent
        PlayerPage {
            Component.onCompleted: root.playerActive = true
            Component.onDestruction: root.playerActive = false
            onBackRequested: {
                seriesStack.pop(StackView.Immediate)
            }
        }
    }

    // ── Error dialog ──────────────────────────────────────────────────────

    Dialog {
        id: errorDialog
        title: "Error"
        modal: true
        anchors.centerIn: parent
        standardButtons: Dialog.Ok

        property string errorMessage: ""

        background: Rectangle {
            color: "#1a1613"
            radius: 10
            border.color: "#ff6b3533"
            border.width: 1
        }

        contentItem: Text {
            text: errorDialog.errorMessage
            color: "#d4183d"
            font.pixelSize: 14
            wrapMode: Text.WordWrap
        }
    }

    Connections {
        target: backend
        function onErrorOccurred(error) {
            errorDialog.errorMessage = error
            errorDialog.open()
        }
    }
}
