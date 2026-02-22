import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Rectangle {
    id: root

    property bool showBackButton: false
    property string initialQuery: ""

    signal searchRequested(string query)
    signal backClicked()

    color: "#0f1923"

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 16
        anchors.rightMargin: 16
        spacing: 16

        // Back button (optional)
        Button {
            visible: root.showBackButton
            text: "←"
            font.pixelSize: 20
            onClicked: root.backClicked()

            background: Rectangle {
                color: parent.hovered ? "#2a3f54" : "transparent"
                radius: 4
            }

            contentItem: Text {
                text: parent.text
                color: "#ff6b35"
                font: parent.font
                horizontalAlignment: Text.AlignHCenter
                verticalAlignment: Text.AlignVCenter
            }
        }

        // Logo/Title
        Text {
            text: "🍿 p2pollo"
            font.pixelSize: 20
            font.weight: Font.Bold
            color: "#ff6b35"
            visible: !root.showBackButton

            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: root.backClicked()
            }
        }

        // Search input
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 40
            color: "#2a3f54"
            radius: 20
            border.color: searchInput.activeFocus ? "#ff6b35" : "transparent"
            border.width: 2

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 16
                anchors.rightMargin: 16
                spacing: 8

                Text {
                    text: "🔍"
                    font.pixelSize: 16
                    color: "#888"
                }

                TextField {
                    id: searchInput
                    Layout.fillWidth: true
                    placeholderText: "Buscar películas..."
                    text: root.initialQuery
                    font.pixelSize: 14
                    color: "#e0e0e0"

                    background: Rectangle {
                        color: "transparent"
                    }

                    onAccepted: {
                        if (text.trim() !== "") {
                            root.searchRequested(text.trim())
                        }
                    }

                    // Style placeholder text
                    placeholderTextColor: "#888"
                }

                // Search button
                Button {
                    visible: searchInput.text !== ""
                    text: "→"
                    font.pixelSize: 16
                    onClicked: {
                        if (searchInput.text.trim() !== "") {
                            root.searchRequested(searchInput.text.trim())
                        }
                    }

                    background: Rectangle {
                        color: parent.hovered ? "#ff8555" : "#ff6b35"
                        radius: 16
                        implicitWidth: 32
                        implicitHeight: 32
                    }

                    contentItem: Text {
                        text: parent.text
                        color: "#fff"
                        font: parent.font
                        horizontalAlignment: Text.AlignHCenter
                        verticalAlignment: Text.AlignVCenter
                    }
                }
            }
        }
    }
}
