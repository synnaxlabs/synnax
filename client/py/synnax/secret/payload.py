#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Annotated, Literal
from uuid import uuid4

from pydantic import BaseModel, Field

from synnax.secret.types_gen import Key


class NewStored(BaseModel):
    """A secret whose value the Core encrypts and stores. No client can read the value
    back.

    :param key: Replaces the secret with this key when it exists.
    """

    key: Key = Field(default_factory=uuid4)
    name: str
    source: Literal["stored"] = "stored"
    value: str


class NewFile(BaseModel):
    """A secret whose value is a file in the secrets folder of the rack that uses it.

    :param key: Replaces the secret with this key when it exists.
    """

    key: Key = Field(default_factory=uuid4)
    name: str
    source: Literal["file"] = "file"
    file_name: str


New = Annotated[NewStored | NewFile, Field(discriminator="source")]
