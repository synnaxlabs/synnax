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

from synnax.access.payload import Policy
from synnax.access.retrieve import PolicyRetriever
from synnax.access.writer import PolicyWriter
from synnax.exceptions import NotFoundError
from synnax.ontology.id import OntologyID


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
                Policy(
                    subjects=subjects,
                    objects=objects,
                    actions=actions,
                )
            ]
        elif isinstance(policies, Policy):
            _policies = [policies]
        else:
            _policies = policies

        created = self._writer.create(_policies)
        return created if isinstance(policies, list) else created[0]

    def retrieve(self, subject: OntologyID) -> Policy | list[Policy]:
        res = self._retriever.retrieve(subject)
        if len(res) > 1:
            return res
        elif len(res) == 1:
            return res[0]
        raise NotFoundError(f"Policy with subject '{subject}' not found.")
