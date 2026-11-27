#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Annotated
from uuid import UUID, uuid4

import pytest
from pydantic import AfterValidator, BaseModel, ValidationError

from x import validate


class Holder(BaseModel):
    owner: Annotated[UUID, AfterValidator(validate.non_nil_uuid)]


class TestNonNilUUID:
    def test_accepts_a_uuid(self) -> None:
        """Should accept a UUID that is not nil."""
        key = uuid4()
        assert Holder(owner=key).owner == key

    def test_rejects_the_nil_uuid(self) -> None:
        """Should refuse the nil UUID."""
        with pytest.raises(ValidationError, match="must be non-zero"):
            Holder(owner=UUID(int=0))
