#!/bin/bash

set -e

PYTHON_VERSION="3.11.7"
NUMPY_VERSION="1.24.3"
PYTHON_VERSION_MAJOR_MINOR="3.11"
PYTHON_INSTALL_DIR="$(pwd)/python_install"

# Download and build Python
curl -O https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz
tar xzf Python-${PYTHON_VERSION}.tgz
cd Python-${PYTHON_VERSION}

# Configure Python for static build
./configure --prefix=${PYTHON_INSTALL_DIR} \
    --disable-shared \
    --enable-optimizations \
    --with-ensurepip=no \
    LDFLAGS="-lm"

make -j$(nproc)
make install
cd ..

# Install pip and numpy with our specific Python
LD_LIBRARY_PATH=${PYTHON_INSTALL_DIR}/lib ${PYTHON_INSTALL_DIR}/bin/python3 -m ensurepip
LD_LIBRARY_PATH=${PYTHON_INSTALL_DIR}/lib ${PYTHON_INSTALL_DIR}/bin/python3 -m pip install --upgrade pip
PYTHONPATH=${PYTHON_INSTALL_DIR}/lib/python${PYTHON_VERSION_MAJOR_MINOR}/site-packages \
    LD_LIBRARY_PATH=${PYTHON_INSTALL_DIR}/lib \
    PYTHONHOME=${PYTHON_INSTALL_DIR} \
    ${PYTHON_INSTALL_DIR}/bin/python3 -m pip install numpy==${NUMPY_VERSION}

# Combine libraries
mkdir -p ${PYTHON_INSTALL_DIR}/lib/combined
cd ${PYTHON_INSTALL_DIR}/lib/combined

# Extract object files
ar -x ../libpython${PYTHON_VERSION_MAJOR_MINOR}.a
find ${PYTHON_INSTALL_DIR}/lib/python${PYTHON_VERSION_MAJOR_MINOR}/site-packages/numpy -name "*.a" -exec ar -x {} \;

# Create combined library
ar -qc libpython${PYTHON_VERSION_MAJOR_MINOR}-combined.a *.o
ranlib libpython${PYTHON_VERSION_MAJOR_MINOR}-combined.a

# Store version
echo ${PYTHON_VERSION} > ${PYTHON_INSTALL_DIR}/VERSION

# Cleanup
cd ../../..
rm -rf Python-${PYTHON_VERSION}.tgz Python-${PYTHON_VERSION} get-pip.py