import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "components"

Item {
    id: root

    signal seriesSelected(var series)
    signal searchRequested(string query)

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // Search bar
        SearchBar {
            Layout.fillWidth: true
            Layout.preferredHeight: 60
            placeholderText: "Buscar series..."
            onSearchRequested: function(query) {
                root.searchRequested(query)
            }
        }

        // Content area
        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: "transparent"

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 20
                spacing: 16

                // Title
                Text {
                    text: seriesGrid.count > 0 ? "Series Populares" : (backend.isLoading ? "Cargando..." : "Series Populares")
                    font.pixelSize: 24
                    font.weight: Font.Medium
                    color: "#f5f3f0"
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
