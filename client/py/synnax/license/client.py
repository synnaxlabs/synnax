#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.license.types_gen import License
from synnax.util.send_required import send_required

State = Literal["ok", "missing", "expired"]
"""Whether a license applies to the Core."""


class Info(BaseModel):
    """The Core's license state, its machine fingerprint, and the license that applies.

    :param state: Whether a license applies to the Core.
    :param warning: Set while the state is ok but a change is near or past.
    :param fingerprint: The hashes that identify the Core's machine.
    :param license: The license that applies, if any.
    """

    state: State
    warning: str = ""
    fingerprint: list[str] = Field(default_factory=list)
    license: License | None = None


class _ActivateRequest(BaseModel):
    token: str


_RETRIEVE_ENDPOINT = "/license/retrieve"
_ACTIVATE_ENDPOINT = "/license/activate"


class Client:
    """Client for the license of a Synnax Core."""

    _client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        self._client = transport

    def retrieve(self) -> Info:
        """Retrieves the Core's license state.

        :returns: The state, the machine fingerprint, and the license that applies.
        """
        return send_required(self._client, _RETRIEVE_ENDPOINT, Empty(), Info)

    def activate(self, token: str) -> Info:
        """Activates a license token on the Core.

        :param token: The signed license token.
        :returns: The resulting license state.
        :raises InvalidLicense: If the token cannot be verified or is malformed.
        :raises LicenseHostMismatch: If the token is bound to a different host.
        :raises ExpiredLicense: If the token no longer applies.
        """
        return send_required(
            self._client, _ACTIVATE_ENDPOINT, _ActivateRequest(token=token), Info
        )
