cmake_minimum_required(VERSION 3.21)

project(p2pollo VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

# Find Qt6
find_package(Qt6 6.5 REQUIRED COMPONENTS
    Core
    Quick
    Network
    Multimedia
)

# Find libmpv
set(MPV_INCLUDE_DIR "D:/Temp/mpv-dev/include" CACHE PATH "mpv include directory")
set(MPV_LIBRARY "D:/Temp/mpv-dev/libmpv.dll.a" CACHE FILEPATH "mpv library")

if(EXISTS "${MPV_INCLUDE_DIR}/mpv/client.h" AND EXISTS "${MPV_LIBRARY}")
    message(STATUS "Found libmpv: ${MPV_LIBRARY}")
    include_directories(${MPV_INCLUDE_DIR})
else()
    message(FATAL_ERROR "libmpv not found at ${MPV_INCLUDE_DIR}")
endif()

# Source files
set(SOURCES
    main.cpp
    backend.cpp
    mpvobject.cpp
)

set(HEADERS
    backend.h
    mpvobject.h
)

# QML files (for IDE integration)
set(QML_FILES
    qml/main.qml
    qml/HomePage.qml
    qml/SearchPage.qml
    qml/MovieDetail.qml
    qml/PlayerPage.qml
    qml/SeriesPage.qml
    qml/SeriesDetail.qml
    qml/SeriesSearchPage.qml
    qml/components/MovieCard.qml
    qml/components/MovieGrid.qml
    qml/components/SearchBar.qml
    qml/components/ProgressBar.qml
)

# Qt resources
qt_add_resources(RESOURCES resources.qrc)

# Create executable
qt_add_executable(${PROJECT_NAME}
    ${SOURCES}
    ${HEADERS}
    ${RESOURCES}
)

# Link libraries
target_link_libraries(${PROJECT_NAME} PRIVATE
    Qt6::Core
    Qt6::Quick
    Qt6::Network
    Qt6::Multimedia
    ${MPV_LIBRARY}
)


# Copy libmpv-2.dll to output directory
if(WIN32 AND EXISTS "${CMAKE_SOURCE_DIR}/../lib/libmpv-2.dll")
    add_custom_command(TARGET ${PROJECT_NAME} POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E copy_if_different
            "${CMAKE_SOURCE_DIR}/../lib/libmpv-2.dll"
            "$<TARGET_FILE_DIR:${PROJECT_NAME}>"
        COMMENT "Copying libmpv-2.dll to output directory"
    )
endif()

# Set QML import path for Qt Creator
set(QML_IMPORT_PATH "${CMAKE_SOURCE_DIR}/qml" CACHE STRING "" FORCE)

# Installation
install(TARGETS ${PROJECT_NAME}
    BUNDLE DESTINATION .
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)

# Print configuration
message(STATUS "Qt version: ${Qt6_VERSION}")
message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
