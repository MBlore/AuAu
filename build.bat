@echo off
setlocal EnableDelayedExpansion

REM Check if vcvars64.bat exists.
if not exist "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvars64.bat" (
    echo Couldn't find C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvars64.bat
    echo Please ensure Visual Studio is installed.
    exit /b 1
)

REM Check if NASM is installed and in the PATH.
where nasm >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo NASM not found. Please ensure NASM is installed and in the PATH.
    exit /b 1
)

REM Initialize the Visual Studio environment variables for 64-bit compilation.
call "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvars64.bat"

REM Build the compiler.
cd compiler
go build -o ../AuAu.exe
cd ..

REM Check if the build was successful.
if %ERRORLEVEL% neq 0 (
    echo Build failed.
    exit /b %ERRORLEVEL%
)

REM Compile test source file.
AuAu.exe build test.au

REM Check if the compilation was successful.
if %ERRORLEVEL% neq 0 (
    echo Compilation failed.
    exit /b %ERRORLEVEL%
)

REM Ensure bin folder exists.
if not exist bin (
    mkdir bin
)

REM Remove stale object files so renamed runtime sources do not leave duplicate symbols behind.
del /q bin\*.obj >nul 2>&1

REM Compile the runtime library.
REM Iterate over all .c files in the runtime directory and compile them to object files.
for %%f in (compiler\runtime\win64\*.c) do (
    cl /nologo /MD /c %%f /Fo:bin\%%~nf.obj >nul
    if %ERRORLEVEL% neq 0 (
        echo Compilation of %%f failed.
        exit /b %ERRORLEVEL%
    )
)

REM Compile the assembly and link to executable.
nasm -f win64 out.asm -o bin\out.obj
if %ERRORLEVEL% neq 0 (
    echo NASM assembly failed.
    exit /b %ERRORLEVEL%
)

REM Link all *.obj files in the bin folder.
set OBJLIST=
for %%f in (bin\*.obj) do (
    set OBJLIST=!OBJLIST! bin\%%~nf.obj
)

echo Linking...
link /nologo !OBJLIST! ^
    /SUBSYSTEM:CONSOLE ^
    /ENTRY:mainCRTStartup ^
    ucrt.lib vcruntime.lib msvcrt.lib legacy_stdio_definitions.lib ^
    /OUT:bin\out.exe || (
        echo Linking failed.
        exit /b %ERRORLEVEL%
    )
echo Done. Output: bin\out.exe
