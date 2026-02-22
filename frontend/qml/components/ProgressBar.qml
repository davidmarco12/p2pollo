import QtQuick
import QtQuick.Controls

Rectangle {
    id: root

    property real value: 0.0  // 0.0 to 1.0
    property color barColor: "#ff6b35"
    property color backgroundColor: "#333"

    height: 6
    color: backgroundColor
    radius: height / 2

    Rectangle {
        width: parent.width * Math.max(0, Math.min(1, root.value))
        height: parent.height
        color: root.barColor
        radius: parent.radius

        Behavior on width {
            NumberAnimation { duration: 200; easing.type: Easing.OutCubic }
        }
    }
}
