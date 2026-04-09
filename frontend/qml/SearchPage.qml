import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "components"

Item {
    id: root

    property string searchQuery: ""

    signal movieSelected(var movie)
    signal backRequested()

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // Search bar with back button
        SearchBar {
            Layout.fillWidth: true
            Layout.preferredHeight: 60
            showBackButton: true
            initialQuery: root.searchQuery
            onSearchRequested: function(query) {
                root.searchQuery = query
                backend.searchMovies(query)
            }
            onBackClicked: {
                root.backRequested()
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
                    text: 'Resultados para "' + root.searchQuery + '"'
                    font.pixelSize: 24
                    font.weight: Font.Medium
                    color: "#f5f3f0"
                }

                // Loading indicator or movie grid
                Item {
                    Layout.fillWidth: true
                    Layout.fillHeight: true

                    // Loading spinner
                    BusyIndicator {
                        anchors.centerIn: parent
                        running: backend.isLoading && movieGrid.count === 0
                        visible: running
                    }

                    // No results message
                    Text {
                        anchors.centerIn: parent
                        text: "No se encontraron resultados"
                        font.pixelSize: 16
                        color: "#8a837c"
                        visible: !backend.isLoading && movieGrid.count === 0
                    }

                    // Movie grid
                    MovieGrid {
                        id: movieGrid
                        anchors.fill: parent
                        visible: movieGrid.count > 0
                        onMovieClicked: function(movie) {
                            root.movieSelected(movie)
                        }
                    }
                }
            }
        }
    }

    // Perform search when component is created
    Component.onCompleted: {
        if (root.searchQuery !== "") {
            backend.searchMovies(root.searchQuery)
        }
    }

    // Handle search results
    Connections {
        target: backend
        function onSearchResultsReceived(movies) {
            movieGrid.setMovies(movies)
        }
    }
}
