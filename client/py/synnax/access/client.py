#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
from __future__ import annotations

import uuid

from pydantic import PrivateAttr

from synnax.access.payload import PolicyPayload, OntologyID
from synnax.access.retrieve import PolicyRetriever
from synnax.access.writer import PolicyWriter
from synnax.exceptions import NotFoundError


class Policy(PolicyPayload):
    __client: PolicyClient | None = PrivateAttr(None)

    def __init__(
        self,
        *,
        key: uuid.UUID | None = None,
        subjects: list[OntologyID],
        objects: list[OntologyID],
        actions: list[str],
        _client: PolicyClient | None = None,
    ) -> None:
        super().__init__(
            key=key,
            subjects=subjects,
            objects=objects,
            actions=actions,
        )
        self.__client = _client

    def to_payload(self) -> PolicyPayload:
        return PolicyPayload(
            key=self.key,
            subjects=self.subjects,
            objects=self.objects,
            actions=self.actions,
        )


class PolicyClient:
    _retriever: PolicyRetriever
    _writer: PolicyWriter

    def __init__(
        self,
        retriever: PolicyRetriever,
        creator: PolicyWriter,
    ):
        self._retriever = retriever
        self._writer = creator

    def delete(self, keys: uuid.UUID|list[uuid.UUID]) -> None:
        self._writer.delete([keys] if isinstance(keys, uuid.UUID) else keys)

    def create(
        self,
        policies: Policy | list[Policy] | None = None,
        *,
        subjects: list[OntologyID] = None,
        objects: list[OntologyID] = None,
        actions: list[str] = None,
    ) -> Policy | list[Policy]:
        if policies is None:
            _policies = [
                PolicyPayload(
                    subjects=subjects,
                    objects=objects,
                    actions=actions,
                )
            ]
        elif isinstance(policies, Policy):
            _policies = [policies.to_payload()]
        else:
            _policies = [p.to_payload() for p in policies]

        created = self.__sugar(self._writer.create(_policies))
        return created if isinstance(policies, list) else created[0]

    def retrieve(self, subject: OntologyID) -> Policy | list[Policy]:
        res = self._retriever.retrieve(subject)
        if len(res) > 1:
            return res
        elif len(res) == 1:
            return res[0]
        raise NotFoundError(f"Policy with subject '{subject}' not found.")

    def __sugar(self, policies: list[PolicyPayload]) -> list[Policy]:
        return [Policy(**p.dict()) for p in policies]
