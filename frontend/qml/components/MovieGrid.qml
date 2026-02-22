import QtQuick
import QtQuick.Controls

ScrollView {
    id: root

    property var movies: []
    readonly property int count: moviesModel.count

    signal movieClicked(var movie)

    // Public method to update movies
    function setMovies(moviesArray) {
        moviesModel.clear()
        for (var i = 0; i < moviesArray.length; i++) {
            moviesModel.append(moviesArray[i])
        }
    }

    clip: true

    // Movies model
    ListModel {
        id: moviesModel
    }

    // Grid view
    GridView {
        id: gridView
        anchors.fill: parent
        cellWidth: 180
        cellHeight: 300
        model: moviesModel

        delegate: MovieCard {
            width: gridView.cellWidth - 20
            height: gridView.cellHeight - 20
            movie: model
            onClicked: {
                root.movieClicked(model)
            }
        }

        // Smooth scrolling
        ScrollBar.vertical: ScrollBar {
            policy: ScrollBar.AsNeeded
        }
    }
}
