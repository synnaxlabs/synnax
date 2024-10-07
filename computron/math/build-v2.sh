#!/bin/bash

set -e

# Set up variables
PYTHON_VERSION="3.9.13"
NUMPY_VERSION="1.21.6"
PYTHON_INSTALL_DIR="$(pwd)/python_install"
GO_FILE="main.go"

# Download and build Python
curl -O https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz
tar xzf Python-${PYTHON_VERSION}.tgz
cd Python-${PYTHON_VERSION}

# Configure Python for static build
./configure --prefix=${PYTHON_INSTALL_DIR} \
    --disable-shared \
    --enable-static \
    --with-ensurepip=no \
    LDFLAGS="-Wl,-rpath,${PYTHON_INSTALL_DIR}/lib"

# Build Python
make -j$(sysctl -n hw.ncpu)

# Install Python
make install

cd ..

# Install pip (required for numpy installation)
curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
${PYTHON_INSTALL_DIR}/bin/python3 get-pip.py

# Install numpy
${PYTHON_INSTALL_DIR}/bin/pip3 install numpy==${NUMPY_VERSION}

# Combine static libraries
mkdir -p ${PYTHON_INSTALL_DIR}/lib/combined
cd ${PYTHON_INSTALL_DIR}/lib/combined

# Extract object files from Python static library
ar -x ../libpython3.9.a

# Extract object files from NumPy static libraries
numpy_lib_path=$(find ${PYTHON_INSTALL_DIR}/lib/python3.9/site-packages/numpy -name '*.a')
for lib in $numpy_lib_path; do
    ar -x $lib
done

# Create combined static library
ar -qc libpython3.9-combined.a *.o
ranlib libpython3.9-combined.a

cd ../../..

# Set up environment variables for CGO
export CGO_CFLAGS="-I${PYTHON_INSTALL_DIR}/include/python3.9 -I${PYTHON_INSTALL_DIR}/lib/python3.9/site-packages/numpy/core/include"
export CGO_LDFLAGS="-L${PYTHON_INSTALL_DIR}/lib/combined -lpython3.9-combined -ldl -framework CoreFoundation -framework SystemConfiguration"

# Build the Go program
go build -ldflags="-extldflags=-Wl,-sectcreate,__TEXT,__info_plist,Info.plist" -o python_go_app ${GO_FILE}

# Run the program
./python_go_app

# Clean up
rm -rf Python-${PYTHON_VERSION} Python-${PYTHON_VERSION}.tgz get-pip.py
