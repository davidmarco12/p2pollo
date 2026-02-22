import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: root
    width: 1024
    height: 768
    visible: true
    title: "p2pollo - P2P Streaming"

    color: "#1b2636"

    // View states
    property string currentView: "home"
    property var selectedMovie: null

    // Stack view for navigation
    StackView {
        id: stackView
        anchors.fill: parent
        initialItem: homePageComponent

        // Smooth transitions
        pushEnter: Transition {
            PropertyAnimation {
                property: "opacity"
                from: 0
                to: 1
                duration: 200
            }
        }
        pushExit: Transition {
            PropertyAnimation {
                property: "opacity"
                from: 1
                to: 0
                duration: 200
            }
        }
    }

    // HomePage component
    Component {
        id: homePageComponent
        HomePage {
            onMovieSelected: function(movie) {
                selectedMovie = movie
                stackView.push(movieDetailComponent)
            }
            onSearchRequested: function(query) {
                stackView.push(searchPageComponent, { "searchQuery": query })
            }
        }
    }

    // SearchPage component
    Component {
        id: searchPageComponent
        SearchPage {
            onMovieSelected: function(movie) {
                selectedMovie = movie
                stackView.push(movieDetailComponent)
            }
            onBackRequested: {
                stackView.pop()
            }
        }
    }

    // MovieDetail component
    Component {
        id: movieDetailComponent
        MovieDetail {
            movie: selectedMovie
            onPlayRequested: function(magnetLink) {
                stackView.push(playerPageComponent, { "magnetLink": magnetLink })
            }
            onBackRequested: {
                stackView.pop()
            }
        }
    }

    // PlayerPage component
    Component {
        id: playerPageComponent
        PlayerPage {
            onBackRequested: {
                stackView.pop()
            }
        }
    }

    // Error dialog
    Dialog {
        id: errorDialog
        title: "Error"
        modal: true
        anchors.centerIn: parent
        standardButtons: Dialog.Ok

        property string errorMessage: ""

        contentItem: Text {
            text: errorDialog.errorMessage
            color: "#f44336"
            wrapMode: Text.WordWrap
        }
    }

    // Connect to backend error signal
    Connections {
        target: backend
        function onErrorOccurred(error) {
            errorDialog.errorMessage = error
            errorDialog.open()
        }
    }
}
