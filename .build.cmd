@echo off
rem Build helper:
rem   1. Read productVersion from wails.json (single source of truth)
rem   2. Add Go / Node / NSIS to PATH so Wails CLI can find them
rem   3. Run `wails build -clean -nsis -o opensca-ui-<version>.exe`
rem
rem Output filenames produced:
rem   build\bin\opensca-ui-<version>.exe                (single exe)
rem   build\bin\opensca-ui-<version>-amd64-installer.exe (NSIS installer)
rem
rem To release: bump productVersion in wails.json — both filenames follow.
for /f "usebackq delims=" %%v in (`powershell -NoProfile -Command "(Get-Content 'wails.json' ^| ConvertFrom-Json).info.productVersion"`) do set "VER=%%v"
echo Building OpenSCA UI v%VER%...
if not exist "internal\bundle\opensca-cli.exe" (
    echo WARNING: internal\bundle\opensca-cli.exe missing ^|^| run scripts\fetch-cli.ps1
) else (
    for %%B in ("internal\bundle\opensca-cli.exe") do if %%~zB LSS 1048576 echo WARNING: opensca-cli.exe looks like a placeholder ^(^<1MB^) ^|^| run scripts\fetch-cli.ps1
)
set PATH=D:\App\NSIS;D:\App\Go\bin;D:\App\NodeJS;%PATH%
cd /d "f:\MyCode\opensca-ui"
"C:\Users\hdec\go\bin\wails.exe" build -clean -nsis -o "opensca-ui-%VER%.exe"
echo wails-exit=%ERRORLEVEL%