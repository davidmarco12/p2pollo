import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Rectangle {
    id: root

    property var movie: null

    signal clicked()

    width: 160
    height: 280
    color: "transparent"

    ColumnLayout {
        anchors.fill: parent
        spacing: 8

        // Poster with hover effect
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 240
            color: "#2a3f54"
            radius: 8
            clip: true

            Image {
                anchors.fill: parent
                source: movie ? movie.posterUrl : ""
                fillMode: Image.PreserveAspectCrop
                asynchronous: true
                smooth: true

                // Loading placeholder
                Rectangle {
                    anchors.fill: parent
                    color: "#2a3f54"
                    visible: parent.status !== Image.Ready
                    z: -1
                }
            }

            // Rating badge
            Rectangle {
                anchors.top: parent.top
                anchors.right: parent.right
                anchors.margins: 8
                width: 50
                height: 24
                color: "#ffc107"
                radius: 4
                visible: movie && movie.rating > 0

                Text {
                    anchors.centerIn: parent
                    text: movie ? "★ " + movie.rating.toFixed(1) : ""
                    font.pixelSize: 11
                    font.weight: Font.Bold
                    color: "#000"
                }
            }

            // Hover overlay
            Rectangle {
                anchors.fill: parent
                color: "#000"
                opacity: mouseArea.containsMouse ? 0.3 : 0
                Behavior on opacity { NumberAnimation { duration: 150 } }
            }

            // Click area
            MouseArea {
                id: mouseArea
                anchors.fill: parent
                hoverEnabled: true
                onClicked: root.clicked()
                cursorShape: Qt.PointingHandCursor
            }

            // Scale animation on hover
            scale: mouseArea.containsMouse ? 1.05 : 1.0
            Behavior on scale {
                NumberAnimation { duration: 150; easing.type: Easing.OutCubic }
            }
        }

        // Title and year
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 2

            Text {
                text: movie ? movie.title : ""
                font.pixelSize: 13
                font.weight: Font.Medium
                color: "#e0e0e0"
                wrapMode: Text.WordWrap
                maximumLineCount: 2
                elide: Text.ElideRight
                Layout.fillWidth: true
            }

            Text {
                text: movie ? movie.year : ""
                font.pixelSize: 11
                color: "#888"
            }
        }
    }
}
