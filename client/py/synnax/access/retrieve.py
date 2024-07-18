#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Protocol, cast

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient
from synnax.access.payload import Policy
from synnax.ontology.id import OntologyID


class _Request(Payload):
    subject: OntologyID


class _Response(Payload):
    policies: list[Policy] | None


class PolicyRetriever:
    __ENDPOINT = "/access/policy/retrieve"
    __client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.__client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def retrieve(self, subject: OntologyID) -> list[Policy]:
        return self.__execute(
            _Request(subject=subject)
        )

    def __execute(
        self,
        req: _Request,
    ) -> list[Policy]:
        res, exc = self.__client.send(self.__ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        return res.policies
