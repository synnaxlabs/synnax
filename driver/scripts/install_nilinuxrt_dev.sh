# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

bazel-6.4.0 build --stamp //core/pkg/version:version --define=platform=nilinuxrt
bazel-6.4.0 build //driver --define=platform=nilinuxrt
./bazel-bin/driver/driver stop
sudo ./bazel-bin/driver/driver install
./bazel-bin/driver/driver start
