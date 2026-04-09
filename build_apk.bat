@echo off
set JAVA_HOME=D:\Android\jbr
set PATH=D:\Android\jbr\bin;%PATH%
echo JAVA_HOME=%JAVA_HOME%
cd /d D:\Proyectos\p2pollo\android
D:\Proyectos\p2pollo\android\gradlew.bat assembleDebug
