bazel-6.4.0 build //driver:driver_main --define=platform=nilinuxrt
sudo systemctl stop synnax-driver.service
sudo ./bazel-bin/driver/driver_main install
sudo systemctl restart synnax-driver.service
tail -f /var/log/synnax-driver.log