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

    def test_import_names_from_file(self, client: sy.Synnax, tmp_path: Path) -> None:
        """A nameless envelope is named after the file, extension stripped."""
        name = f"imex-file-{uuid.uuid4()}"
        envelope = json.loads(_log_envelope_json("unused"))
        del envelope["name"]
        path = tmp_path / f"{name}.json"
        path.write_text(json.dumps(envelope))
        id = client.imex.import_(path)
        out = tmp_path / "out.json"
        client.imex.export(id, out)
        assert json.loads(out.read_bytes())["name"] == name

    def test_import_body_name_wins(self, client: sy.Synnax, tmp_path: Path) -> None:
        """A name in the envelope body takes precedence over the file name."""
        name = f"imex-body-{uuid.uuid4()}"
        path = tmp_path / "Some Other Name.json"
        path.write_text(_log_envelope_json(name))
        id = client.imex.import_(path)
        out = tmp_path / "out.json"
        client.imex.export(id, out)
        assert json.loads(out.read_bytes())["name"] == name

    def test_import_with_parent(self, client: sy.Synnax, tmp_path: Path) -> None:
        """The imported resource is parented under the given ontology resource."""
        proj = client.projects.create(name=f"imex-proj-{uuid.uuid4()}")
        path = tmp_path / "in.json"
        path.write_text(_log_envelope_json(f"imex-parent-{uuid.uuid4()}"))
        id = client.imex.import_(path, parent=sy.project.ontology_id(proj.key))
        children = client.ontology.retrieve_children(sy.project.ontology_id(proj.key))
        assert id.key in [child.id.key for child in children]

    def test_import_with_nonexistent_parent(
        self, client: sy.Synnax, tmp_path: Path
    ) -> None:
        """Importing under a parent that does not exist fails."""
        path = tmp_path / "in.json"
        path.write_text(_log_envelope_json(f"imex-orphan-{uuid.uuid4()}"))
        with pytest.raises(sy.NotFoundError):
            client.imex.import_(
                path, parent=sy.ontology.ID(type="project", key=str(uuid.uuid4()))
            )

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
