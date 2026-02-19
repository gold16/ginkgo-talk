@echo off
echo Building Ginkgo Talk...
go build -ldflags "-H windowsgui" -o GinkgoTalk.exe .
if %errorlevel% equ 0 (
    echo Build successful: GinkgoTalk.exe
) else (
    echo Build failed!
)
pause
