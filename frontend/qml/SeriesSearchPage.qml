import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "components"

Item {
    id: root

    property string searchQuery: ""

    signal seriesSelected(var series)
    signal backRequested()

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        SearchBar {
            Layout.fillWidth: true
            Layout.preferredHeight: 60
            showBackButton: true
            initialQuery: root.searchQuery
            placeholderText: "Buscar series..."
            onSearchRequested: function(query) {
                root.searchQuery = query
                backend.searchSeries(query)
            }
            onBackClicked: {
                root.backRequested()
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: "transparent"

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 20
                spacing: 16

                Text {
                    text: 'Resultados para "' + root.searchQuery + '"'
                    font.pixelSize: 24
                    font.weight: Font.Medium
                    color: "#f5f3f0"
                }

                Item {
                    Layout.fillWidth: true
                    Layout.fillHeight: true

                    BusyIndicator {
                        anchors.centerIn: parent
                        running: backend.isLoading && seriesGrid.count === 0
                        visible: running
                    }

                    Text {
                        anchors.centerIn: parent
                        text: "No se encontraron resultados"
                        font.pixelSize: 16
                        color: "#8a837c"
                        visible: !backend.isLoading && seriesGrid.count === 0
                    }

                    MovieGrid {
                        id: seriesGrid
                        anchors.fill: parent
                        isSeries: true
                        visible: seriesGrid.count > 0
                        onMovieClicked: function(series) {
                            root.seriesSelected(series)
                        }
                    }
                }
            }
        }
    }

    Component.onCompleted: {
        if (root.searchQuery !== "") {
            backend.searchSeries(root.searchQuery)
        }
    }

    Connections {
        target: backend
        function onSeriesSearchResultsReceived(series) {
            seriesGrid.setMovies(series)
        }
    }
}
