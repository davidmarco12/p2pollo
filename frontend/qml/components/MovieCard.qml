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
            color: "#1a1613"
            radius: 10
            clip: true

            Image {
                anchors.fill: parent
                source: movie ? movie.posterUrl : ""
                fillMode: Image.PreserveAspectCrop
                asynchronous: true
                cache: true
                smooth: true
                mipmap: true

                // Loading placeholder
                Rectangle {
                    anchors.fill: parent
                    color: "#1a1613"
                    visible: parent.status !== Image.Ready
                    z: -1

                    // Loading indicator
                    BusyIndicator {
                        anchors.centerIn: parent
                        running: parent.visible && parent.parent.status === Image.Loading
                        width: 32
                        height: 32
                    }
                }
            }

            // Rating badge
            Rectangle {
                anchors.top: parent.top
                anchors.right: parent.right
                anchors.margins: 8
                width: 50
                height: 24
                color: "#ff6b3533"
                radius: 6
                border.color: "#ff6b35"
                border.width: 1
                visible: movie && movie.rating > 0

                RowLayout {
                    anchors.centerIn: parent
                    spacing: 2

                    Text {
                        text: "★"
                        font.pixelSize: 12
                        color: "#ff6b35"
                    }

                    Text {
                        text: movie ? movie.rating.toFixed(1) : ""
                        font.pixelSize: 11
                        font.weight: Font.Medium
                        color: "#ff6b35"
                    }
                }
            }

            // Hover overlay with gradient
            Rectangle {
                anchors.fill: parent
                gradient: Gradient {
                    GradientStop { position: 0.0; color: "#00000000" }
                    GradientStop { position: 0.6; color: "#66000000" }
                    GradientStop { position: 1.0; color: "#cc000000" }
                }
                opacity: mouseArea.containsMouse ? 1.0 : 0
                Behavior on opacity { NumberAnimation { duration: 300 } }

                // Play icon on hover
                Text {
                    anchors.centerIn: parent
                    text: "▶"
                    font.pixelSize: 48
                    color: "#ff6b35"
                    opacity: mouseArea.containsMouse ? 1.0 : 0
                    Behavior on opacity { NumberAnimation { duration: 300 } }
                }
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
                color: "#f5f3f0"
                wrapMode: Text.WordWrap
                maximumLineCount: 2
                elide: Text.ElideRight
                Layout.fillWidth: true
            }

            Text {
                text: movie ? movie.year : ""
                font.pixelSize: 11
                color: "#8a837c"
            }
        }
    }
}
