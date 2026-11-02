#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest
from pydantic import ValidationError

from framework import utils


class TestReadJSONObject:
    def test_should_read_a_nested_object(self, tmp_path: Path) -> None:
        path = tmp_path / "export.json"
        path.write_text('{"type": "lineplot", "channels": {"y1": [1, 2]}}')
        assert utils.read_json_object(str(path)) == {
            "type": "lineplot",
            "channels": {"y1": [1, 2]},
        }

    @pytest.mark.parametrize("content", ["[1, 2]", '"text"', "3", "null"])
    def test_should_reject_json_that_is_not_an_object(
        self, tmp_path: Path, content: str
    ) -> None:
        path = tmp_path / "export.json"
        path.write_text(content)
        with pytest.raises(ValidationError):
            utils.read_json_object(str(path))
