#!/bin/bash

set -e

# Set up variables
PYTHON_VERSION="3.9.13"
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

# Combine static libraries
mkdir -p ${PYTHON_INSTALL_DIR}/lib/combined
cd ${PYTHON_INSTALL_DIR}/lib/combined

ar -x ../libpython3.9.a

for lib in ../python3.9/config-3.9-darwin/*.a; do
    ar -x $lib
done

ar -qc libpython3.9-combined.a *.o
ranlib libpython3.9-combined.a

cd ../../..

# Set up environment variables for CGO
export CGO_CFLAGS="-I${PYTHON_INSTALL_DIR}/include/python3.9"
export CGO_LDFLAGS="-L${PYTHON_INSTALL_DIR}/lib/combined -lpython3.9-combined -ldl -framework CoreFoundation"

# Build the Go program
go build -ldflags '-extldflags "-static"' -o python_go_app ${GO_FILE}
