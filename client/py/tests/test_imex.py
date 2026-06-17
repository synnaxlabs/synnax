#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import uuid
from pathlib import Path

import pytest

import synnax as sy
from synnax.imex import Envelope


def _log_envelope(name: str) -> Envelope:
    """A minimal valid v1 log envelope, matching core's testdata/import_v1.json."""
    return Envelope.model_validate(
        {
            "version": 1,
            "type": "log",
            "name": name,
            "channels": [
                {
                    "channel": 1,
                    "color": "red",
                    "notation": "scientific",
                    "precision": 2,
                    "alias": "temp",
                }
            ],
            "remote_created": False,
            "timestamp_precision": 1,
            "show_channel_names": True,
            "show_receipt_timestamp": False,
        }
    )


class TestEnvelope:
    """Unit tests for the Envelope type — no server required."""

    def test_roundtrip_preserves_extras(self) -> None:
        env = _log_envelope("rt")
        parsed = Envelope.model_validate_json(env.model_dump_json())
        assert parsed.version == env.version
        assert parsed.type == env.type
        assert parsed.name == env.name
        assert parsed.model_extra == env.model_extra

    def test_promoted_fields_marshal_at_top_level(self) -> None:
        env = Envelope.model_validate(
            {"version": 1, "type": "log", "name": "n", "foo": "bar"}
        )
        dumped = env.model_dump(by_alias=True)
        assert dumped["version"] == 1
        assert dumped["type"] == "log"
        assert dumped["name"] == "n"
        assert dumped["foo"] == "bar"

    def test_rejects_empty_type(self) -> None:
        with pytest.raises(ValueError, match="type"):
            Envelope.model_validate({"version": 1, "type": "", "name": "n"})

    def test_rejects_empty_name(self) -> None:
        with pytest.raises(ValueError, match="name"):
            Envelope.model_validate({"version": 1, "type": "log", "name": ""})


@pytest.mark.imex
class TestImex:
    """Round-trip imex against a live Synnax Core.

    Uses the ``log`` resource type.
    """

    def test_import_envelope_then_export(self, client: sy.Synnax) -> None:
        """Envelope (in-memory) → upload via send; export via send."""
        name = f"imex-rt-{uuid.uuid4()}"
        id = client.imex.import_(_log_envelope(name))
        assert id.type == "log"
        assert uuid.UUID(id.key)
        exported = client.imex.export(id)
        assert exported is not None
        assert exported.type == "log"
        assert exported.name == name
        assert exported.version >= 1

    def test_import_from_path(self, client: sy.Synnax, tmp_path: Path) -> None:
        """Path source → streamed upload."""
        name = f"imex-path-{uuid.uuid4()}"
        path = tmp_path / "in.json"
        path.write_text(_log_envelope(name).model_dump_json())
        id = client.imex.import_(path)
        assert id.type == "log"
        assert uuid.UUID(id.key)
        exported = client.imex.export(id)
        assert exported is not None and exported.name == name

    def test_export_to_path(self, client: sy.Synnax, tmp_path: Path) -> None:
        """Path dest → streamed download; on-disk content parses back."""
        name = f"imex-export-path-{uuid.uuid4()}"
        id = client.imex.import_(_log_envelope(name))
        out = tmp_path / "log.json"
        assert client.imex.export(id, dest=out) is None
        parsed = Envelope.model_validate_json(out.read_bytes())
        assert parsed.name == name and parsed.type == "log"
