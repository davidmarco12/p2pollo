import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "components"

Item {
    id: root

    signal movieSelected(var movie)
    signal searchRequested(string query)

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // Search bar
        SearchBar {
            Layout.fillWidth: true
            Layout.preferredHeight: 60
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
                    text: "Películas Populares"
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

                    // Movie grid
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
        }
    }

    // Load popular movies on component creation
    Component.onCompleted: {
        backend.getPopularMovies()
    }

    // Handle popular movies response
    Connections {
        target: backend
        function onPopularMoviesReceived(movies) {
            movieGrid.setMovies(movies)
        }
    }
}
