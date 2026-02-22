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
                duration: 150
            }
        }
        pushExit: Transition {
            PropertyAnimation {
                property: "opacity"
                from: 1
                to: 0
                duration: 150
            }
        }
        popEnter: Transition {
            PropertyAnimation {
                property: "opacity"
                from: 0
                to: 1
                duration: 150
            }
        }
        popExit: Transition {
            PropertyAnimation {
                property: "opacity"
                from: 1
                to: 0
                duration: 150
            }
        }
    }

    // HomePage component
    Component {
        id: homePageComponent
        HomePage {
            StackView.onActivated: {
                // Refresh if needed
            }
            onMovieSelected: function(movie) {
                selectedMovie = movie
                stackView.push(movieDetailComponent, StackView.Immediate)
            }
            onSearchRequested: function(query) {
                stackView.push(searchPageComponent, { "searchQuery": query }, StackView.Immediate)
            }
        }
    }

    // SearchPage component
    Component {
        id: searchPageComponent
        SearchPage {
            onMovieSelected: function(movie) {
                selectedMovie = movie
                stackView.push(movieDetailComponent, StackView.Immediate)
            }
            onBackRequested: {
                stackView.pop(StackView.Immediate)
            }
        }
    }

    // MovieDetail component
    Component {
        id: movieDetailComponent
        MovieDetail {
            movie: selectedMovie
            onPlayRequested: function(magnetLink) {
                stackView.push(playerPageComponent, { "magnetLink": magnetLink }, StackView.Immediate)
            }
            onBackRequested: {
                stackView.pop(StackView.Immediate)
            }
        }
    }

    // PlayerPage component
    Component {
        id: playerPageComponent
        PlayerPage {
            onBackRequested: {
                stackView.pop(StackView.Immediate)
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

    // Connect to backend error signal
    Connections {
        target: backend
        function onErrorOccurred(error) {
            errorDialog.errorMessage = error
            errorDialog.open()
        }
    }
}
