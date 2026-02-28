import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "components"

Item {
    id: root

    signal seriesSelected(var series)

    property var genreFilters: ["Todos", "Drama", "Comedy", "Crime", "Action", "Sci-Fi", "Thriller", "Animation", "Documentary"]
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
                    text: "Series Populares"
                    font.pixelSize: 28
                    font.weight: Font.Bold
                    color: "#f5f3f0"
                    font.letterSpacing: -0.5
                }

                Text {
                    text: "Las series más vistas del momento"
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
                        color: isActive ? "#00a8e1" : "#171513"
                        border.color: isActive ? "#00a8e1" : "#2a2521"
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

        // Loading indicator or series grid
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            BusyIndicator {
                anchors.centerIn: parent
                running: backend.isLoading && seriesGrid.count === 0
                visible: running
            }

            MovieGrid {
                id: seriesGrid
                anchors.fill: parent
                isSeries: true
                visible: !backend.isLoading || seriesGrid.count > 0
                onMovieClicked: function(series) {
                    root.seriesSelected(series)
                }
            }
        }
    }

    Component.onCompleted: {
        backend.getPopularSeries()
    }

    Connections {
        target: backend
        function onPopularSeriesReceived(series) {
            seriesGrid.setMovies(series)
        }
        function onSeriesSearchResultsReceived(series) {
            seriesGrid.setMovies(series)
        }
    }
}
