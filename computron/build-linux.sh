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

# Configure Python for static build with all symbols exported and explicit architecture
./configure --prefix=${PYTHON_INSTALL_DIR} \
    --disable-shared \
    --enable-optimizations \
    --with-ensurepip=no \
    --build=x86_64-linux-gnu \
    --host=x86_64-linux-gnu \
    CFLAGS="-fPIC" \
    LDFLAGS="-lm -Wl,--export-dynamic"

make -j$(nproc)
make install
cd ..

# Set up environment variables
export PYTHONPATH="${PYTHON_INSTALL_DIR}/lib/python${PYTHON_VERSION_MAJOR_MINOR}/site-packages"
export LD_LIBRARY_PATH="${PYTHON_INSTALL_DIR}/lib"
export PYTHONHOME="${PYTHON_INSTALL_DIR}"
export PYTHON_INCLUDE_DIR="${PYTHON_INSTALL_DIR}/include/python${PYTHON_VERSION_MAJOR_MINOR}"
export PYTHON_LIBRARY="${PYTHON_INSTALL_DIR}/lib/libpython${PYTHON_VERSION_MAJOR_MINOR}.a"

# Install pip using the specific Python build
${PYTHON_INSTALL_DIR}/bin/python3 -m ensurepip
${PYTHON_INSTALL_DIR}/bin/python3 -m pip install --upgrade pip

# Install numpy with specific build flags
CFLAGS="-fPIC" \
NPY_DISABLE_SVML=1 \
${PYTHON_INSTALL_DIR}/bin/python3 -m pip install --no-binary numpy numpy==${NUMPY_VERSION}

# Combine libraries
mkdir -p ${PYTHON_INSTALL_DIR}/lib/combined
cd ${PYTHON_INSTALL_DIR}/lib/combined

# Clean work directory
rm -f *.o *.a

# Extract object files from Python library
ar -x ../libpython${PYTHON_VERSION_MAJOR_MINOR}.a

# Create a temporary directory for numpy objects
mkdir -p numpy_objects
cd numpy_objects

# Extract object files from NumPy static libraries
find ${PYTHON_INSTALL_DIR}/lib/python${PYTHON_VERSION_MAJOR_MINOR}/site-packages/numpy -name "*.a" -exec ar -x {} \;

# Move back to combined directory
cd ..

# Move numpy objects to main directory
mv numpy_objects/*.o .
rmdir numpy_objects

# Create combined library
ar -rc libpython${PYTHON_VERSION_MAJOR_MINOR}-combined.a *.o
ranlib libpython${PYTHON_VERSION_MAJOR_MINOR}-combined.a

# Verify the archive
ar -t libpython${PYTHON_VERSION_MAJOR_MINOR}-combined.a > /dev/null || {
    echo "Error: Combined archive verification failed"
    exit 1
}

# Store version
echo ${PYTHON_VERSION} > ${PYTHON_INSTALL_DIR}/VERSION

# Cleanup
cd ../../..
rm -rf Python-${PYTHON_VERSION}.tgz Python-${PYTHON_VERSION}