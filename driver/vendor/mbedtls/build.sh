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