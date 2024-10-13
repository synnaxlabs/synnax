# Windows Build Script for Python and NumPy with Go

# Set variables
$PYTHON_VERSION = "3.9.13"
$NUMPY_VERSION = "1.21.6"
$PYTHON_INSTALL_DIR = "$PWD\python_install"
$PYTHON_INSTALLER = "python-$PYTHON_VERSION-amd64.exe"
$PYTHON_INSTALLER_URL = "https://www.python.org/ftp/python/$PYTHON_VERSION/$PYTHON_INSTALLER"

# Create installation directory
if (!(Test-Path -Path $PYTHON_INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $PYTHON_INSTALL_DIR
}

# Download Python installer
Write-Host "Downloading Python $PYTHON_VERSION installer..."
Invoke-WebRequest -Uri $PYTHON_INSTALLER_URL -OutFile $PYTHON_INSTALLER

# Install Python
Write-Host "Installing Python to $PYTHON_INSTALL_DIR..."
Start-Process -FilePath ".\$PYTHON_INSTALLER" -ArgumentList "/quiet InstallAllUsers=0 PrependPath=0 Include_pip=1 TargetDir=$PYTHON_INSTALL_DIR" -Wait

# Install NumPy
Write-Host "Installing NumPy $NUMPY_VERSION..."
$PYTHON_EXE = "$PYTHON_INSTALL_DIR\python.exe"
& $PYTHON_EXE -m pip install numpy==$NUMPY_VERSION

# Clean up
Write-Host "Cleaning up..."
Remove-Item $PYTHON_INSTALLER

Write-Host "Python and NumPy installation completed."
