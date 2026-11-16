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

from synnax.certificate.types_gen import FileCertificate, Key


class NewStored(BaseModel):
    """A certificate whose contents the Core stores. The Core reads its details from the
    PEM.

    :param key: Replaces the certificate with this key when it exists.
    :param pem: The certificate chain as PEM, leaf first.
    :param private_key_pem: The private key as PEM. The Core checks that it matches
        the certificate and saves it as a secret.
    """

    key: Key = Field(default_factory=uuid4)
    name: str
    source: Literal["stored"] = "stored"
    pem: str
    private_key_pem: str | None = None


New = Annotated[NewStored | FileCertificate, Field(discriminator="source")]
