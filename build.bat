@echo off
echo Building Ginkgo Talk...
echo Embedding icon resource...
rsrc -ico app.ico -o rsrc_windows.syso
go build -ldflags "-H windowsgui" -o GinkgoTalk.exe .
if %errorlevel% equ 0 (
    echo Build successful: GinkgoTalk.exe
) else (
    echo Build failed!
)
pause
