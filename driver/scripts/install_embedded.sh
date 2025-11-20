bazel build -c opt --config=hide_symbols //driver:driver
mv ./bazel-bin/driver/driver ./core/pkg/service/driver/assets/driver
