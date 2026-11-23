#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from pathlib import Path

from pydantic import BaseModel

ATTRIBUTES = (
    Path(__file__).resolve().parents[3]
    / "oracle"
    / "plugin"
    / "py"
    / "keywords"
    / "model_attributes.txt"
)


def test_oracle_escapes_every_base_model_attribute() -> None:
    """Should list in Oracle every public BaseModel attribute a field could shadow."""
    meta = type(BaseModel)
    public = {
        name
        for name in set(dir(BaseModel)) | set(dir(meta))
        if not name.startswith("_") and hasattr(BaseModel, name)
    }
    assert set(ATTRIBUTES.read_text().split()) == public
