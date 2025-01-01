#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient, send_required, Empty

from synnax.ranger.payload import RangeKey, RangePayload
from synnax.ontology.payload import ID


class _CreateRequest(Payload):
    parent: ID | None
    ranges: list[RangePayload]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[RangeKey]


_CREATE_ENDPOINT = "/range/create"
_DELETE_ENDPOINT = "/range/delete"


class RangeWriter:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug", "range.create")
    def create(
        self, ranges: list[RangePayload], *, parent: ID | None = None
    ) -> list[RangePayload]:
        req = _CreateRequest(ranges=ranges, parent=parent)
        return send_required(
            self._client, _CREATE_ENDPOINT, req, _CreateResponse
        ).ranges

    @trace("debug", "range.delete")
    def delete(self, keys: list[RangeKey]):
        req = _DeleteRequest(keys=keys)
        send_required(self._client, _DELETE_ENDPOINT, req, Empty)
