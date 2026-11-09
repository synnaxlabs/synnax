#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A fake `gh` executable for tests that run repository scripts."""

import json
import subprocess
import sys
from fnmatch import fnmatchcase
from pathlib import Path
from typing import Any

_ROUTES = "routes.json"
_CALLS = "calls.jsonl"
_FAILURES = "failures.json"


class FakeGH:
    """A fake `gh`. Put `bin_dir` first on the `PATH` of the script under test.

    Records each call, then answers it from the served routes. A call that matches no
    route fails.
    """

    def __init__(self, path: Path) -> None:
        self.bin_dir = path / "bin"
        self.bin_dir.mkdir()
        self._routes: dict[str, list[Any]] = {}
        (self.bin_dir / _ROUTES).write_text("{}")
        (self.bin_dir / _FAILURES).write_text("{}")
        (self.bin_dir / _CALLS).touch()
        gh = self.bin_dir / "gh"
        gh.write_text(
            f'#!/usr/bin/env bash\nexec "{sys.executable}" "{__file__}" '
            f'"{self.bin_dir}" "$@"\n'
        )
        gh.chmod(0o755)

    def serve(self, pattern: str, *pages: Any) -> None:
        """Answers calls whose first two arguments match the glob `pattern`, such as
        `api */pulls/7` or `pr merge`.

        Prints each JSON page through `jq -r` with the call's `--jq` expression, so a
        call with no pages prints nothing. The earliest served pattern wins. Serving a
        pattern again replaces its pages.
        """
        self._routes[pattern] = list(pages)
        (self.bin_dir / _ROUTES).write_text(json.dumps(self._routes))

    def fail(self, pattern: str, message: str, *, times: int = 1) -> None:
        """Fails the next `times` calls that match `pattern`, printing `message` to
        stderr. A failure comes before any served route."""
        failures = json.loads((self.bin_dir / _FAILURES).read_text())
        failures[pattern] = [message, times]
        (self.bin_dir / _FAILURES).write_text(json.dumps(failures))

    def calls(self) -> list[list[str]]:
        """Returns the arguments of each call so far, oldest first."""
        lines = (self.bin_dir / _CALLS).read_text().splitlines()
        return [json.loads(line) for line in lines]


def _answer(bin_dir: Path, args: list[str]) -> None:
    with (bin_dir / _CALLS).open("a") as calls:
        calls.write(json.dumps(args) + "\n")
    call = " ".join(args[:2])
    failures = json.loads((bin_dir / _FAILURES).read_text())
    for pattern, (message, times) in failures.items():
        if times > 0 and fnmatchcase(call, pattern):
            failures[pattern][1] = times - 1
            (bin_dir / _FAILURES).write_text(json.dumps(failures))
            sys.exit(message)
    for pattern, pages in json.loads((bin_dir / _ROUTES).read_text()).items():
        if fnmatchcase(call, pattern):
            expression = args[args.index("--jq") + 1] if "--jq" in args else "."
            for page in pages:
                subprocess.run(
                    ["jq", "-r", expression],
                    input=json.dumps(page),
                    text=True,
                    check=True,
                )
            return
    sys.exit(f"unexpected gh call: {args}")


if __name__ == "__main__":
    _answer(Path(sys.argv[1]), sys.argv[2:])
