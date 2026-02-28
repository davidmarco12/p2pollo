import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Rectangle {
    id: root

    property var movie: null
    property bool isSeries: false

    signal clicked()

    width: 160
    height: 290
    color: "transparent"

    ColumnLayout {
        anchors.fill: parent
        spacing: 8

        // Poster
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 240
            color: "#171513"
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

                Rectangle {
                    anchors.fill: parent
                    color: "#171513"
                    visible: parent.status !== Image.Ready
                    z: -1

                    BusyIndicator {
                        anchors.centerIn: parent
                        running: parent.visible && parent.parent.status === Image.Loading
                        width: 28
                        height: 28
                    }
                }
            }

            // Badge TV (series)
            Rectangle {
                anchors.top: parent.top
                anchors.left: parent.left
                anchors.margins: 8
                width: 30
                height: 18
                color: "#00a8e1"
                radius: 4
                visible: root.isSeries

                Text {
                    anchors.centerIn: parent
                    text: "TV"
                    font.pixelSize: 9
                    font.weight: Font.Bold
                    color: "#fff"
                }
            }

            // Rating badge
            Rectangle {
                anchors.top: parent.top
                anchors.right: parent.right
                anchors.margins: 8
                width: ratingRow.width + 12
                height: 22
                color: "#0a090844"
                radius: 6
                border.color: root.isSeries ? "#00a8e155" : "#ff6b3555"
                border.width: 1
                visible: movie && movie.rating > 0

                Row {
                    id: ratingRow
                    anchors.centerIn: parent
                    spacing: 3

                    Text {
                        text: "★"
                        font.pixelSize: 10
                        color: root.isSeries ? "#00a8e1" : "#ff6b35"
                    }

                    Text {
                        text: movie ? movie.rating.toFixed(1) : ""
                        font.pixelSize: 10
                        font.weight: Font.SemiBold
                        color: root.isSeries ? "#00a8e1" : "#ff6b35"
                    }
                }
            }

            // Hover overlay
            Rectangle {
                anchors.fill: parent
                gradient: Gradient {
                    GradientStop { position: 0.0; color: "#00000000" }
                    GradientStop { position: 0.5; color: "#55000000" }
                    GradientStop { position: 1.0; color: "#cc000000" }
                }
                opacity: mouseArea.containsMouse ? 1.0 : 0
                Behavior on opacity { NumberAnimation { duration: 250 } }

                Text {
                    anchors.centerIn: parent
                    text: "▶"
                    font.pixelSize: 44
                    color: "#fff"
                    opacity: mouseArea.containsMouse ? 0.9 : 0
                    Behavior on opacity { NumberAnimation { duration: 250 } }
                }
            }

            MouseArea {
                id: mouseArea
                anchors.fill: parent
                hoverEnabled: true
                onClicked: root.clicked()
                cursorShape: Qt.PointingHandCursor
            }

            scale: mouseArea.containsMouse ? 1.04 : 1.0
            Behavior on scale {
                NumberAnimation { duration: 180; easing.type: Easing.OutCubic }
            }
        }

        // Título y año
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 2

            Text {
                text: movie ? movie.title : ""
                font.pixelSize: 12
                font.weight: Font.Medium
                color: "#e8e4e0"
                wrapMode: Text.WordWrap
                maximumLineCount: 2
                elide: Text.ElideRight
                Layout.fillWidth: true
                lineHeight: 1.3
            }

            Text {
                text: movie ? movie.year : ""
                font.pixelSize: 11
                color: "#5a5551"
            }
        }
    }
}
