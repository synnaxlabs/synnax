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
mkdir open62541/build && cd open62541/build
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo -DCMAKE_INSTALL_PREFIX=../out -DUA_ENABLE_ENCRYPTION=MBEDTLS ..
make
make install
