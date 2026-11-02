#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from x.os import _get_linux_distribution


class TestGetLinuxDistribution:
    def test_falls_back_to_os_release(self, tmp_path: Path) -> None:
        """Should read /etc/os-release when lsb_release is not installed."""
        os_release = tmp_path / "os-release"
        os_release.write_text('NAME="Ubuntu"\nPRETTY_NAME="Ubuntu 24.04 LTS"\n')
        name = _get_linux_distribution(
            lsb_release="synnax-missing-lsb-release", os_release=os_release
        )
        assert name == "Ubuntu 24.04 LTS"
