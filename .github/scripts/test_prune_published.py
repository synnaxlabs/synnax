#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
import json
import sys
import urllib.error
from pathlib import Path

import pytest

import prune_published


@pytest.fixture(autouse=True)
def clear_published_files_cache() -> None:
    prune_published.published_files.cache_clear()


class TestParseNameVersion:
    """Tests for wheel and sdist file name parsing."""

    @pytest.mark.parametrize(
        ("filename", "name", "version"),
        [
            ("alamos-0.56.0-py3-none-any.whl", "alamos", "0.56.0"),
            ("synnax_x-0.56.0-py3-none-any.whl", "synnax-x", "0.56.0"),
            ("synnax_freighter-0.56.0-py3-none-any.whl", "synnax-freighter", "0.56.0"),
            ("Some_Pkg-1.0-1-py3-none-any.whl", "some-pkg", "1.0"),
            ("alamos-0.56.0.tar.gz", "alamos", "0.56.0"),
            ("synnax_freighter-0.56.0.tar.gz", "synnax-freighter", "0.56.0"),
            ("synnax-0.56.1.tar.gz", "synnax", "0.56.1"),
        ],
    )
    def test_should_parse_normalized_name_and_version(
        self, filename: str, name: str, version: str
    ) -> None:
        """Should extract the PEP 503 normalized name and the version."""
        assert prune_published.parse_name_version(filename) == (name, version)

    @pytest.mark.parametrize(
        "filename",
        [
            ".gitignore",
            "notes.txt",
            "foo-py3-none-any.whl",
            "synnax-x-0.56.0-py3-none-any.whl",
            "foo.tar.gz",
        ],
    )
    def test_should_reject_malformed_file_names(self, filename: str) -> None:
        """Should raise instead of querying a nonexistent release."""
        with pytest.raises(ValueError):
            prune_published.parse_name_version(filename)


class TestPublishedFiles:
    """Tests for the PyPI release lookup."""

    def test_should_return_file_names_for_existing_release(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should collect the file names published for the release."""
        body = json.dumps(
            {"urls": [{"filename": "alamos-0.56.0-py3-none-any.whl"}]}
        ).encode()
        monkeypatch.setattr(
            prune_published.urllib.request, "urlopen", lambda url: io.BytesIO(body)
        )
        assert prune_published.published_files("alamos", "0.56.0") == frozenset(
            {"alamos-0.56.0-py3-none-any.whl"}
        )

    def test_should_return_empty_set_for_missing_release(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should treat a 404 as an unpublished release."""

        def raise_not_found(url: str) -> io.BytesIO:
            raise urllib.error.HTTPError(url, 404, "Not Found", None, None)  # type: ignore[arg-type]

        monkeypatch.setattr(prune_published.urllib.request, "urlopen", raise_not_found)
        assert prune_published.published_files("alamos", "9.9.9") == frozenset()

    def test_should_propagate_other_http_errors(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should not swallow non-404 failures."""

        def raise_server_error(url: str) -> io.BytesIO:
            raise urllib.error.HTTPError(url, 503, "Unavailable", None, None)  # type: ignore[arg-type]

        monkeypatch.setattr(
            prune_published.urllib.request, "urlopen", raise_server_error
        )
        with pytest.raises(urllib.error.HTTPError):
            prune_published.published_files("alamos", "0.56.0")


class TestMain:
    """Tests for the pruning entry point."""

    def test_should_prune_published_and_keep_new_files(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should delete already-published files and keep unpublished ones."""
        (tmp_path / "alamos-0.56.0-py3-none-any.whl").touch()
        (tmp_path / "alamos-0.56.0.tar.gz").touch()
        (tmp_path / "synnax-0.56.1-py3-none-any.whl").touch()
        (tmp_path / ".gitignore").touch()
        published = {
            ("alamos", "0.56.0"): frozenset(
                {"alamos-0.56.0-py3-none-any.whl", "alamos-0.56.0.tar.gz"}
            )
        }
        monkeypatch.setattr(
            prune_published,
            "published_files",
            lambda name, version: published.get((name, version), frozenset()),
        )
        monkeypatch.setattr(sys, "argv", ["prune_published.py", str(tmp_path)])
        prune_published.main()
        assert sorted(p.name for p in tmp_path.iterdir()) == [
            ".gitignore",
            "synnax-0.56.1-py3-none-any.whl",
        ]

    def test_should_keep_missing_file_of_partially_published_release(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should retry the sdist when only the wheel made it to PyPI."""
        (tmp_path / "alamos-0.56.0-py3-none-any.whl").touch()
        (tmp_path / "alamos-0.56.0.tar.gz").touch()
        monkeypatch.setattr(
            prune_published,
            "published_files",
            lambda name, version: frozenset({"alamos-0.56.0-py3-none-any.whl"}),
        )
        monkeypatch.setattr(sys, "argv", ["prune_published.py", str(tmp_path)])
        prune_published.main()
        assert [p.name for p in tmp_path.iterdir()] == ["alamos-0.56.0.tar.gz"]

    def test_should_exit_when_dist_directory_is_missing(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should fail loudly when the dist directory does not exist."""
        monkeypatch.setattr(
            sys, "argv", ["prune_published.py", str(tmp_path / "nonexistent")]
        )
        with pytest.raises(SystemExit):
            prune_published.main()
