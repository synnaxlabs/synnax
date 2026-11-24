#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fakes of `gh` and Vercel Blob for tests that run repository scripts."""

import json
import subprocess
import sys
import threading
import urllib.parse
from dataclasses import dataclass
from fnmatch import fnmatchcase
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

_ROUTES = "routes.json"
_CALLS = "calls.jsonl"
_FAILURES = "failures.json"
_INPUTS = "inputs.jsonl"


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
        (self.bin_dir / _INPUTS).touch()
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

    def inputs(self) -> list[str | None]:
        """Returns the body each call so far read through `--input`, oldest first, or
        None for a call without one."""
        lines = (self.bin_dir / _INPUTS).read_text().splitlines()
        return [json.loads(line) for line in lines]


def _input(args: list[str]) -> str | None:
    if "--input" not in args:
        return None
    path = args[args.index("--input") + 1]
    return sys.stdin.read() if path == "-" else Path(path).read_text()


@dataclass(frozen=True)
class BlobPut:
    """One upload that a FakeBlob received."""

    pathname: str
    headers: dict[str, str]
    body: bytes


class FakeBlob:
    """A local Vercel Blob API. Point the script under test at `url`.

    Records each upload and answers it with a public URL under `PUBLIC`, or with the
    next status in `failures`. Call `close` when done.
    """

    PUBLIC = "https://blob.test"

    def __init__(self) -> None:
        self.puts: list[BlobPut] = []
        self.failures: list[int] = []
        fake = self

        class Handler(BaseHTTPRequestHandler):
            def do_PUT(self) -> None:
                query = urllib.parse.urlparse(self.path).query
                pathname = urllib.parse.parse_qs(query)["pathname"][0]
                body = self.rfile.read(int(self.headers["Content-Length"]))
                headers = {k.lower(): v for k, v in self.headers.items()}
                status = fake.failures.pop(0) if fake.failures else 200
                if status == 200:
                    fake.puts.append(BlobPut(pathname, headers, body))
                reply = json.dumps({"url": f"{fake.PUBLIC}/{pathname}"}).encode()
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(reply)))
                self.end_headers()
                self.wfile.write(reply)

            def log_message(self, format: str, *args: Any) -> None:
                pass

        self._server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.url = f"http://127.0.0.1:{self._server.server_port}"
        threading.Thread(target=self._server.serve_forever, daemon=True).start()

    def close(self) -> None:
        self._server.shutdown()
        self._server.server_close()


def _answer(bin_dir: Path, args: list[str]) -> None:
    with (bin_dir / _CALLS).open("a") as calls:
        calls.write(json.dumps(args) + "\n")
    with (bin_dir / _INPUTS).open("a") as inputs:
        inputs.write(json.dumps(_input(args)) + "\n")
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
