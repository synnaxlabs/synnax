#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import json
import uuid
from pathlib import Path

import pytest

import synnax as sy


def _log_envelope_json(name: str) -> str:
    """A minimal valid v1 log envelope, matching core's testdata/import_v1.json."""
    return json.dumps(
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


@pytest.mark.imex
class TestImex:
    """Round-trip imex against a live Synnax Core.

    Uses the ``log`` resource type.
    """

    def test_import_valid(self, client: sy.Synnax, tmp_path: Path) -> None:
        """Path source → streamed upload."""
        name = f"imex-path-{uuid.uuid4()}"
        path = tmp_path / "in.json"
        path.write_text(_log_envelope_json(name))
        id = client.imex.import_(path)
        assert id.type == "log"
        assert uuid.UUID(id.key)

    def test_import_invalid(self, client: sy.Synnax, tmp_path: Path) -> None:
        """An envelope with an unrecognized type is rejected."""
        path = tmp_path / "in.json"
        path.write_text(
            json.dumps({"version": 1, "type": "not_a_real_type", "name": "bad"})
        )
        with pytest.raises(sy.ValidationError):
            client.imex.import_(path)

    def test_export(self, client: sy.Synnax, tmp_path: Path) -> None:
        """Path dest → streamed download; on-disk content parses back."""
        name = f"imex-export-path-{uuid.uuid4()}"
        src = tmp_path / "in.json"
        src.write_text(_log_envelope_json(name))
        id = client.imex.import_(src)
        out = tmp_path / "log.json"
        client.imex.export(id, out)
        parsed = json.loads(out.read_bytes())
        assert parsed["name"] == name and parsed["type"] == "log"

    def test_export_nonexistent(self, client: sy.Synnax, tmp_path: Path) -> None:
        """Exporting a resource that does not exist raises NotFoundError."""
        id = sy.ontology.ID(type="log", key=str(uuid.uuid4()))
        out = tmp_path / "log.json"
        with pytest.raises(sy.NotFoundError):
            client.imex.export(id, out)
