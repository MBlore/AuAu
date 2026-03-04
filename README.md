# Dependencies

## Visual Studio

Any version of Microsoft Visual Studio is needed for the compiler tools.

At minimum: https://visualstudio.microsoft.com/vs/community/

Make sure you install "Desktop development with C++" to install the C toolchain.

The VS toolchain is used for linking .obj files and producing a compiled Windows EXE.

We also use the toolchain for compiling a C-Runtime shim that interfaces with the Windows API.

## NASM

You'll also need "NASM". On window "winget install NASM.NASM" and make sure nasm.exe is in PATH.

NASM compiles assembly source into .obj files ready for the linker.

# Building

Just run "./build.bat" - it builds the compiler, then runs the compiler on the source file 'test.au' and produces 'out.exe'.

That 'out.exe' is a complete stand-alone windows binary.

It will check and warn you if you don't have the Visual Studio toolchain or NASM installed.
