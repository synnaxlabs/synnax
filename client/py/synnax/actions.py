#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Mapping, Sequence
from typing import Any

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.transport import send_batch
from x.lists import normalize


class DispatchRequest[K, A](BaseModel):
    """Applies actions to one record, in order, in one transaction.

    Attributes:
        key: Is the key of the record.
        dispatch_key: Names the dispatch in the actions that the Core broadcasts.
        actions: Are the actions to apply.
    """

    key: K
    dispatch_key: str = ""
    actions: list[A]


class Sender[K, A: BaseModel]:
    """Sends the actions of one record type to the dispatch route of the type. Each
    call applies all of its actions in one transaction, or none of them.
    """

    _client: UnaryClient
    _endpoint: str
    _update: type[A]

    def __init__(self, *, client: UnaryClient, endpoint: str, update: type[A]) -> None:
        """
        :param client: The client to send with.
        :param endpoint: The dispatch route of the record type, such as
            ``/rack/dispatch``.
        :param update: The update action of the record type.
        """
        self._client = client
        self._endpoint = endpoint
        self._update = update

    def dispatch(self, key: K, actions: A | list[A]) -> None:
        """Applies actions to the record with the given key, in order.

        :param key: The key of the record.
        :param actions: The actions to apply.
        """
        self._send([DispatchRequest(key=key, actions=normalize(actions))])

    def dispatch_each(self, keys: Sequence[K], actions: A | list[A]) -> None:
        """Applies the same actions to each record with one of the given keys.

        :param keys: The keys of the records.
        :param actions: The actions to apply to each record.
        """
        acts = normalize(actions)
        self._send([DispatchRequest(key=k, actions=acts) for k in keys])

    def update(self, updates: Sequence[Mapping[str, Any]]) -> None:
        """Changes records by partial records. A None in an update clears its field.

        :param updates: The partial records. Each holds the key of the record that it
            changes.
        :raises ValidationError: If an update holds a value of the wrong type.
        """
        reqs: list[DispatchRequest[K, A]] = []
        for u in updates:
            value = dict(u)
            key = value.pop("key")
            action = self._update.model_validate({"update": {"value": value}})
            reqs.append(DispatchRequest(key=key, actions=[action]))
        self._send(reqs)

    def _send(self, reqs: list[DispatchRequest[K, A]]) -> None:
        if len(reqs) == 1:
            self._client.send(self._endpoint, reqs[0], Empty)
        elif reqs:
            send_batch(self._client, self._endpoint, reqs)
