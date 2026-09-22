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
    """The Core's license state, its host fingerprint, and the license that applies.

    Attributes:
        state: Whether a license applies to the Core.
        warning: Set while the state is ok but a change is near or past.
        fingerprint: The hashes that identify the Core's host.
        license: The license that applies, if any.
    """

    state: State
    warning: str = ""
    fingerprint: list[str] = Field(default_factory=list)
    license: License | None = None


class _InfoResponse(BaseModel):
    state: State
    warning: str = ""
    host: list[str] = Field(default_factory=list)
    grant: License | None = None

    def info(self) -> Info:
        return Info(
            state=self.state,
            warning=self.warning,
            fingerprint=self.host,
            license=self.grant,
        )


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

        Returns:
            The license state, the host fingerprint, and the license that applies.
        """
        return send_required(
            self._client, _RETRIEVE_ENDPOINT, Empty(), _InfoResponse
        ).info()

    def activate(self, token: str) -> Info:
        """Activates a license token on the Core.

        Args:
            token: The signed license token.

        Returns:
            The resulting license state.

        Raises:
            InvalidLicense: If the token cannot be verified or does not fit this Core.
            LicenseHostError: If the token is bound to a different host.
            ExpiredLicense: If the token no longer applies.
        """
        return send_required(
            self._client,
            _ACTIVATE_ENDPOINT,
            _ActivateRequest(token=token),
            _InfoResponse,
        ).info()
