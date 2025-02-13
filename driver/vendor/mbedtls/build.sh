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

curl -LO https://github.com/Mbed-TLS/mbedtls/releases/download/v3.6.0/mbedtls-3.6.0.tar.bz2
tar -xjf mbedtls-3.6.0.tar.bz2
mv mbedtls-3.6.0 mbedtls
cd mbedtls
# ./scripts/make_generated_files.bat
cd ..
cmake -G "Visual Studio 17 2022" -S mbedtls -B mbedtls-build
cmake --build mbedtls-build --config Release
cmake --install mbedtls-build --config Release
cmake --install mbedtls-build --config Release --prefix mbedtls-install
