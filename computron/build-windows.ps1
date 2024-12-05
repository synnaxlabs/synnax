# Requires Visual Studio Build Tools to be installed
# Run this script in PowerShell with Administrator privileges

$ErrorActionPreference = "Stop"

# Configuration
$PYTHON_VERSION = "3.11.7"
$PYTHON_VERSION_MAJOR_MINOR = $PYTHON_VERSION -replace '^(\d+\.\d+).*','$1'
$PYTHON_INSTALL_DIR = Join-Path $PWD "python_install"
$BUILD_DIR = Join-Path $PWD "Python-$PYTHON_VERSION"

# Ensure required tools are available
function Check-Requirements {
    Write-Host "Checking requirements..."

    if (-not (Test-Path "${env:ProgramFiles}\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat")) {
        throw "Visual Studio Build Tools not found. Please install Visual Studio 2022 with C++ build tools."
    }
}

# Download and extract Python source
function Get-PythonSource {
    Write-Host "Downloading Python $PYTHON_VERSION source..."
    $url = "https://www.python.org/ftp/python/$PYTHON_VERSION/Python-$PYTHON_VERSION.tgz"
    $output = "Python-$PYTHON_VERSION.tgz"

    Invoke-WebRequest -Uri $url -OutFile $output
    tar -xf $output
}

# Configure and build Python
function Build-Python {
    Write-Host "Building Python..."

    Push-Location $BUILD_DIR

    # Create PCbuild directory if it doesn't exist
    New-Item -ItemType Directory -Path "PCbuild" -Force | Out-Null
    Push-Location "PCbuild"

    # Get external dependencies
    Write-Host "Getting external dependencies..."
    cmd /c "call `"${env:ProgramFiles}\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat`" amd64 && .\get_externals.bat"

    # Create custom props file
    $customProps = @"
<?xml version="1.0" encoding="utf-8"?>
<Project xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <PropertyGroup>
    <DisableStdExtensionLibs>true</DisableStdExtensionLibs>
    <RuntimeLibrary>MultiThreaded</RuntimeLibrary>
  </PropertyGroup>
  <ItemDefinitionGroup>
    <ClCompile>
      <RuntimeLibrary>MultiThreaded</RuntimeLibrary>
      <PreprocessorDefinitions>_USING_V110_SDK71_;%(PreprocessorDefinitions)</PreprocessorDefinitions>
    </ClCompile>
    <Link>
      <AdditionalDependencies>libucrt.lib;libvcruntime.lib;%(AdditionalDependencies)</AdditionalDependencies>
      <IgnoreSpecificDefaultLibraries>libucrt.lib;%(IgnoreSpecificDefaultLibraries)</IgnoreSpecificDefaultLibraries>
    </Link>
  </ItemDefinitionGroup>
</Project>
"@
    $customProps | Out-File -FilePath "custom.props" -Encoding UTF8

    # Create build script
    $buildScript = @"
@echo off
setlocal
call "${env:ProgramFiles}\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat" amd64
set CL=/MT
MSBuild pcbuild.proj /t:Build /p:Configuration=Release /p:Platform=x64 ^
    /p:IncludeExternals=true ^
    /p:IncludeCTypes=true ^
    /p:IncludeSSL=true ^
    /p:IncludeTkinter=true ^
    /p:UseMultiToolTask=true ^
    /p:ForceImportBeforeCppTargets=%cd%\custom.props
endlocal
"@

    $buildScript | Out-File -FilePath "build_python.bat" -Encoding ASCII

    # Run the build
    Write-Host "Building Python (this may take a while)..."
    cmd /c build_python.bat

    Pop-Location
    Pop-Location
}

# Install Dependencies and Set Up Directory Structure
function Install-Dependencies {
    Write-Host "Setting up directory structure..."

    # Create Mac-like directory structure
    $includePath = "$PYTHON_INSTALL_DIR\include\python$PYTHON_VERSION_MAJOR_MINOR"
    $libPath = "$PYTHON_INSTALL_DIR\lib\python$PYTHON_VERSION_MAJOR_MINOR"
    $libCombinedPath = "$PYTHON_INSTALL_DIR\lib\combined"

    New-Item -ItemType Directory -Force -Path @(
        $includePath,
        "$includePath\cpython",
        "$includePath\internal",
        $libPath,
        $libCombinedPath,
        "$PYTHON_INSTALL_DIR\bin"
    ) | Out-Null

    # Copy header files maintaining structure
    Write-Host "Copying headers..."
    Copy-Item "$BUILD_DIR\Include\*" $includePath -Recurse -Force
    Copy-Item "$BUILD_DIR\PC\pyconfig.h" $includePath

    # Copy internal headers
    Get-ChildItem "$BUILD_DIR\Include\internal" -Filter "*.h" -Recurse |
        ForEach-Object {
            Copy-Item $_.FullName "$includePath\internal" -Force
        }

    # Copy library files
    Write-Host "Copying libraries..."
    $buildPath = "$BUILD_DIR\PCbuild\amd64\Release"
    if (-not (Test-Path $buildPath)) {
        $buildPath = "$BUILD_DIR\PCbuild\amd64"
    }

    if (Test-Path $buildPath) {
        # Copy to lib/combined
        Get-ChildItem -Path $buildPath -Filter "*.lib" | ForEach-Object {
            Copy-Item $_.FullName $libCombinedPath -Force
        }

        # Also copy to main lib directory
        Get-ChildItem -Path $buildPath -Filter "*.lib" | ForEach-Object {
            Copy-Item $_.FullName "$PYTHON_INSTALL_DIR\lib" -Force
        }
    } else {
        throw "Build directory not found: $buildPath"
    }

    # Copy Python standard library
    Write-Host "Copying Python standard library..."
    Copy-Item "$BUILD_DIR\Lib\*" $libPath -Recurse -Force

    # Save version info
    $PYTHON_VERSION | Out-File -FilePath (Join-Path $PYTHON_INSTALL_DIR "VERSION")
}

# Cleanup temporary files and extracted source
function Cleanup {
    Write-Host "Cleaning up..."
    if (Test-Path $BUILD_DIR) {
        Remove-Item -Path $BUILD_DIR -Recurse -Force
    }
    Remove-Item "Python-$PYTHON_VERSION.tgz" -Force -ErrorAction SilentlyContinue
}

# Main execution
try {
    Check-Requirements
    Get-PythonSource
    Build-Python
    Install-Dependencies
    Cleanup
    Write-Host "Build completed successfully!"
} catch {
    Write-Error "Build failed: $_"
    exit 1
}
