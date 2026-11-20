#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes tests/durations.json, the test times that --shard balances by, from the test
results of CI runs. Download a few recent runs first:

    gh run download <run-id> -p 'test-results-*' -D runs/<run-id>
    uv run python scripts/update_durations.py runs
"""

import argparse
import json
from pathlib import Path

from framework import shards


def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("bundles", type=Path, help="Directory of downloaded results")
    durations = shards.record(parser.parse_args().bundles)
    text = json.dumps(durations, indent=2) + "\n"
    shards.DURATIONS.write_text(text, encoding="utf-8")
    print(f"Wrote {len(durations)} tests to {shards.DURATIONS}")


if __name__ == "__main__":
    main()
