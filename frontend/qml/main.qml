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

        // Tab bar superior — oculta cuando el player está activo
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 44
            visible: !root.playerActive
            color: "#0d0c0b"

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 16
                spacing: 0

                Repeater {
                    model: ["Películas", "Series"]

                    delegate: Rectangle {
                        Layout.preferredWidth: 120
                        Layout.fillHeight: true
                        color: "transparent"

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
                            font.weight: mainContent.currentIndex === index ? Font.Bold : Font.Normal
                            color: mainContent.currentIndex === index ? "#ff6b35" : "#8a837c"
                        }

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: mainContent.currentIndex = index
                        }
                    }
                }

                Item { Layout.fillWidth: true }
            }

            Rectangle {
                anchors.bottom: parent.bottom
                width: parent.width
                height: 1
                color: "#1a1613"
            }
        }

        // Contenido: dos StackViews independientes, uno por tab
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
            onSearchRequested: function(query) {
                moviesStack.push(moviesSearchPageComponent, { "searchQuery": query }, StackView.Immediate)
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
            onSearchRequested: function(query) {
                seriesStack.push(seriesSearchPageComponent, { "searchQuery": query }, StackView.Immediate)
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
