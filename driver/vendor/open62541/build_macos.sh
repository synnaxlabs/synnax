rm -r ./open62541/out
rm -r ./open62541/build
mkdir open62541/build && cd open62541/build
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo -DUA_NAMESPACE_ZERO=FULL -DCMAKE_INSTALL_PREFIX=../out -DUA_ENABLE_ENCRYPTION=MBEDTLS ..
make
make install
