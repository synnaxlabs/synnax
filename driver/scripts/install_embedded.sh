bazel build -c opt --config=hide_symbols //driver:driver
mv ./bazel-bin/driver/driver ./synnax/pkg/service/hardware/embedded/assets/driver