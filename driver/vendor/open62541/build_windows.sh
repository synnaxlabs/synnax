#
# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.
#

rm -r ./open62541/out
rm -r ./open62541/build

# Create build directory
mkdir -p open62541/build && cd open62541/build

# Define MBEDTLS_DIR and run cmake
MBEDTLS_DIR="C:\\Program Files (x86)\\Mbed TLS"
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo \
      -DUA_NAMESPACE_ZERO=FULL \
      -DCMAKE_INSTALL_PREFIX="../out" \
      -DUA_ENABLE_ENCRYPTION=MBEDTLS \
      -DMBEDTLS_LIBRARY="$MBEDTLS_DIR\\lib\\mbedtls.lib" \
      -DMBEDX509_LIBRARY="$MBEDTLS_DIR\\lib\\mbedx509.lib" \
      -DMBEDCRYPTO_LIBRARY="$MBEDTLS_DIR\\lib\\mbedcrypto.lib" \
      -DMBEDTLS_INCLUDE_DIRS="$MBEDTLS_DIR\\include" \
      -DCMAKE_OSX_ARCHITECTURES=x86_64 ..

# Build and install with verbose output
cmake --build . --config RelWithDebInfo --target install --verbose
