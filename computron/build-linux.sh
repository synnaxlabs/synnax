#!/bin/bash

set -e

# Update versions to match macOS build
PYTHON_VERSION="3.11.7"
NUMPY_VERSION="1.24.3"
PYTHON_VERSION_MAJOR_MINOR=$(echo $PYTHON_VERSION | cut -d. -f1-2)
PYTHON_A_FILE="libpython${PYTHON_VERSION_MAJOR_MINOR}.a"
COMBINED_PYTHON_A_FILE="libpython${PYTHON_VERSION_MAJOR_MINOR}-combined.a"
PYTHON_INSTALL_DIR="$(pwd)/python_install"

echo "Python Installation Starting"
echo "Python Version: ${PYTHON_VERSION}"
echo "NumPy Version: ${NUMPY_VERSION}"
echo "Python Install Directory: ${PYTHON_INSTALL_DIR}"
sleep 1

# Download and build Python
curl -O https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz
tar xzf Python-${PYTHON_VERSION}.tgz
cd Python-${PYTHON_VERSION}

# Configure Python for static build
./configure --prefix=${PYTHON_INSTALL_DIR} \
    --disable-shared \
    --enable-optimizations \
    --with-ensurepip=no \
    LDFLAGS="-Wl,-rpath,${PYTHON_INSTALL_DIR}/lib"

# Build Python using all available cores
make -j$(nproc)

# Install Python
make install

cd ..

# Install pip (required for NumPy installation)
curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
${PYTHON_INSTALL_DIR}/bin/python3 get-pip.py

# Install NumPy
${PYTHON_INSTALL_DIR}/bin/pip3 install numpy==${NUMPY_VERSION}

# Combine static libraries
mkdir -p ${PYTHON_INSTALL_DIR}/lib/combined
cd ${PYTHON_INSTALL_DIR}/lib/combined

# Extract object files from Python static library
ar -x ../${PYTHON_A_FILE}

# Extract object files from NumPy static libraries
numpy_lib_path=$(find ${PYTHON_INSTALL_DIR}/lib/python${PYTHON_VERSION_MAJOR_MINOR}/site-packages/numpy -name '*.a')
for lib in $numpy_lib_path; do
    ar -x $lib
done

# Create combined static library
ar -qc ${COMBINED_PYTHON_A_FILE} *.o
ranlib ${COMBINED_PYTHON_A_FILE}

# Create VERSION file
echo ${PYTHON_VERSION} > ${PYTHON_INSTALL_DIR}/VERSION

cd ../../..

# Cleanup
rm Python-${PYTHON_VERSION}.tgz
rm -r Python-${PYTHON_VERSION}
rm get-pip.py

echo "Build and installation completed successfully."