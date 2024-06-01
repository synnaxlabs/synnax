rm -r ./open62541/out
rm -r ./open62541/build
mkdir open62541/build && cd open62541/build
MBEDTLS_DIR="C:\Program Files (x86)\Mbed TLS"
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo -DUA_NAMESPACE_ZERO=FULL -DCMAKE_INSTALL_PREFIX=../out -DUA_ENABLE_ENCRYPTION=MBEDTLS -DMBEDTLS_LIBRARY="C:\Program Files (x86)\Mbed TLS\lib\mbedtls.lib" -DMBEDX509_LIBRARY="C:\Program Files (x86)\Mbed TLS\lib\mbedx509.lib" -DMBEDCRYPTO_LIBRARY="C:\Program Files (x86)\Mbed TLS\lib\mbedcrypto.lib" -DMBEDTLS_INCLUDE_DIRS="C:\Program Files (x86)\Mbed TLS\include" -DCMAKE_OSX_ARCHITECTURES=x86_64 ..
cmake --build . --config RelWithDebInfo --target install
