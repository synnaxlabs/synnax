#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.license.types_gen import Info
from synnax.util.send_required import send_required


class _ActivateRequest(BaseModel):
    key: str


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

    def activate(self, key: str) -> Info:
        """Activates a license key on the Core.

        :param key: The signed license key.
        :returns: The resulting license state.
        :raises InvalidLicense: If the key cannot be verified or is malformed.
        :raises LicenseFingerprintMismatch: If the key is bound to another machine.
        :raises ExpiredLicense: If the key no longer applies.
        """
        return send_required(
            self._client, _ACTIVATE_ENDPOINT, _ActivateRequest(key=key), Info
        )
