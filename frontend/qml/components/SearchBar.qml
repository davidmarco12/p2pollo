import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Rectangle {
    id: root

    property bool showBackButton: false
    property string initialQuery: ""
    property string placeholderText: "Buscar películas..."

    signal searchRequested(string query)
    signal backClicked()

    color: "#1a1613"

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
                color: parent.hovered ? "#2a2521" : "transparent"
                radius: 10
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
        RowLayout {
            visible: !root.showBackButton
            spacing: 12

            Image {
                source: "qrc:/assets/images/output-estesi.png"
                Layout.preferredWidth: 40
                Layout.preferredHeight: 40
                fillMode: Image.PreserveAspectFit
                smooth: true
            }

            Text {
                text: "p2pollo"
                font.pixelSize: 20
                font.weight: Font.Bold
                color: "#f5f3f0"
            }

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
            color: "#2a2521"
            radius: 20
            border.color: searchInput.activeFocus ? "#ff6b35" : "#ff6b3533"
            border.width: 1

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 16
                anchors.rightMargin: 16
                spacing: 8

                Text {
                    text: "🔍"
                    font.pixelSize: 16
                    color: "#8a837c"
                }

                TextField {
                    id: searchInput
                    Layout.fillWidth: true
                    placeholderText: root.placeholderText
                    text: root.initialQuery
                    font.pixelSize: 14
                    color: "#f5f3f0"

                    background: Rectangle {
                        color: "transparent"
                    }

                    onAccepted: {
                        if (text.trim() !== "") {
                            root.searchRequested(text.trim())
                        }
                    }

                    // Style placeholder text
                    placeholderTextColor: "#8a837c"
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
                        color: parent.hovered ? "#ff8f4f" : "#ff6b35"
                        radius: 16
                        implicitWidth: 32
                        implicitHeight: 32
                    }

                    contentItem: Text {
                        text: parent.text
                        color: "#0a0908"
                        font.pixelSize: parent.font.pixelSize
                        font.weight: Font.Medium
                        horizontalAlignment: Text.AlignHCenter
                        verticalAlignment: Text.AlignVCenter
                    }
                }
            }
        }
    }
}
