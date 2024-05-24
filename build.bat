@echo off

go install github.com/tc-hib/go-winres@latest
if '%errorlevel%' neq '0' goto fail

pushd %~dp0 && (go-winres simply --icon remoter.ico & popd)
if '%errorlevel%' neq '0' goto fail

pushd %~dp0 && (go build -ldflags "-H=windowsgui" & popd)
if '%errorlevel%' neq '0' goto fail

echo [SUCCESS]
exit /b 0

:fail
echo [FAIL]
exit /b 1
