import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "components"

Item {
    id: root

    signal movieSelected(var movie)

    property var genreFilters: ["Todos", "Action", "Comedy", "Drama", "Horror", "Sci-Fi", "Thriller", "Animation", "Documentary"]
    property string activeGenre: "Todos"

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 28
        anchors.topMargin: 24
        spacing: 0

        // Sección header
        RowLayout {
            Layout.fillWidth: true
            Layout.bottomMargin: 16
            spacing: 0

            ColumnLayout {
                spacing: 2

                Text {
                    text: "Populares"
                    font.pixelSize: 28
                    font.weight: Font.Bold
                    color: "#f5f3f0"
                    font.letterSpacing: -0.5
                }

                Text {
                    text: "Las películas más vistas del momento"
                    font.pixelSize: 13
                    color: "#8a837c"
                }
            }

            Item { Layout.fillWidth: true }
        }

        // Chips de género
        ScrollView {
            Layout.fillWidth: true
            Layout.preferredHeight: 34
            Layout.bottomMargin: 20
            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
            ScrollBar.vertical.policy: ScrollBar.AlwaysOff
            clip: true

            Row {
                spacing: 8

                Repeater {
                    model: root.genreFilters

                    delegate: Rectangle {
                        property bool isActive: root.activeGenre === modelData
                        width: chipLabel.width + 20
                        height: 30
                        color: isActive ? "#ff6b35" : "#171513"
                        border.color: isActive ? "#ff6b35" : "#2a2521"
                        border.width: 1
                        radius: 15

                        Text {
                            id: chipLabel
                            anchors.centerIn: parent
                            text: modelData
                            font.pixelSize: 12
                            font.weight: isActive ? Font.SemiBold : Font.Normal
                            color: isActive ? "#0a0908" : "#8a837c"
                        }

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.activeGenre = modelData
                        }
                    }
                }
            }
        }

        // Loading indicator or movie grid
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            BusyIndicator {
                anchors.centerIn: parent
                running: backend.isLoading && movieGrid.count === 0
                visible: running
            }

            MovieGrid {
                id: movieGrid
                anchors.fill: parent
                visible: !backend.isLoading || movieGrid.count > 0
                onMovieClicked: function(movie) {
                    root.movieSelected(movie)
                }
            }
        }
    }

    Component.onCompleted: {
        backend.getPopularMovies()
    }

    Connections {
        target: backend
        function onPopularMoviesReceived(movies) {
            movieGrid.setMovies(movies)
        }
    }
}
