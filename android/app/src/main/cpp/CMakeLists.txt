cmake_minimum_required(VERSION 3.22.1)
project("mpvjni")

# libmpv.so precompilada
add_library(mpv SHARED IMPORTED)
set_target_properties(mpv PROPERTIES
    IMPORTED_LOCATION ${CMAKE_SOURCE_DIR}/../jniLibs/${ANDROID_ABI}/libmpv.so
)

# libavcodec.so — necesaria para av_jni_set_java_vm (registra JVM para vo/gpu/android)
add_library(avcodec SHARED IMPORTED)
set_target_properties(avcodec PROPERTIES
    IMPORTED_LOCATION ${CMAKE_SOURCE_DIR}/../jniLibs/${ANDROID_ABI}/libavcodec.so
)

# Headers de mpv
include_directories(${CMAKE_SOURCE_DIR}/include)

add_library(mpvjni SHARED mpvjni.cpp)

target_link_libraries(mpvjni mpv avcodec android log)
