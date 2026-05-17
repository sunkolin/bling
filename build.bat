@echo off
echo ========================================
echo   Bling Game Modifier - Build Script
echo ========================================
echo.

echo [1/3] Cleaning old files...
if exist bling.exe del /f /q bling.exe
if exist bling_simple.exe del /f /q bling_simple.exe
if exist bling_test.exe del /f /q bling_test.exe
if exist bling_new.exe del /f /q bling_new.exe
if exist bling_final.exe del /f /q bling_final.exe
if exist bling_v2.exe del /f /q bling_v2.exe
echo Done cleaning
echo.

echo [2/3] Building program...
echo Generating resource file with administrator manifest...
rsrc -manifest manifest.xml -o rsrc.syso
if %errorlevel% neq 0 (
    echo Failed to generate resource file!
    pause
    exit /b 1
)
go build -ldflags="-H windowsgui" -o bling.exe .
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)
echo Build completed
echo.

echo [3/3] Verifying file...
if exist bling.exe (
    for %%I in (bling.exe) do set size=%%~zI
    echo bling.exe generated (size: %size% bytes)
) else (
    echo bling.exe not found!
    pause
    exit /b 1
)
echo.

echo ========================================
echo   Build Success!
echo   Executable: bling.exe
echo   Note: Will auto-request administrator privileges on launch
echo ========================================
echo.
pause
