sudo rm -r ./open62541/out
sudo rm -r ./open62541/build
mkdir open62541/build && cd open62541/build
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo -DUA_NAMESPACE_ZERO=FULL -DCMAKE_INSTALL_PREFIX=../out -DCMAKE_OSX_ARCHITECTURES=x86_64 ..
make
make install
