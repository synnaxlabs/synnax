#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "toolchain_ref.sh"

WORKFLOW = """
jobs:
  build:
    steps:
      - uses: actions/checkout@v7
      - run: echo hi
      - uses: actions/setup-go-extra@v1
      - uses: actions/setup-go@v7
      - uses: ./.github/actions/setup-pnpm
"""

COMPOSITE = """
runs:
  using: composite
  steps:
    - uses: pnpm/setup@{pin}
    - run: pnpm install
"""


def toolchain_ref(tmp_path: Path, action: str, pin: str = "v3") -> str:
    (tmp_path / "workflow.yaml").write_text(WORKFLOW)
    composite = tmp_path / ".github" / "actions" / "setup-pnpm"
    composite.mkdir(parents=True, exist_ok=True)
    (composite / "action.yaml").write_text(COMPOSITE.format(pin=pin))
    return subprocess.run(
        [str(SCRIPT), "workflow.yaml", action],
        cwd=tmp_path,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


class TestToolchainRef:
    def test_direct_step(self, tmp_path: Path) -> None:
        assert toolchain_ref(tmp_path, "actions/setup-go") == "actions/setup-go@v7"

    def test_composite_step(self, tmp_path: Path) -> None:
        assert toolchain_ref(tmp_path, "pnpm/setup") == "pnpm/setup@v3"

    def test_composite_pin_change(self, tmp_path: Path) -> None:
        assert toolchain_ref(tmp_path, "pnpm/setup", pin="v4") == "pnpm/setup@v4"

    def test_unused_action(self, tmp_path: Path) -> None:
        assert toolchain_ref(tmp_path, "bazel-contrib/setup-bazel") == ""
