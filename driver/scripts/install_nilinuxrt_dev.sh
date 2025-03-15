bazel-6.4.0 build --stamp //synnax/pkg/version:version --define=platform=nilinuxrt
bazel-6.4.0 build //driver --define=platform=nilinuxrt
./bazel-bin/driver/driver stop
sudo ./bazel-bin/driver/driver install
./bazel-bin/driver/driver start