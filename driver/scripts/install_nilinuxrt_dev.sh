bazel-6.4.0 build --stamp //synnax/pkg/version:version --define=platform=nilinuxrt
bazel-6.4.0 build //driver:driver_main --define=platform=nilinuxrt
./bazel-bin/driver/driver_main stop
sudo ./bazel-bin/driver/driver_main install
./bazel-bin/driver/driver_main start