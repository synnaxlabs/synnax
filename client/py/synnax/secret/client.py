#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.exceptions import ValidationError
from synnax.secret.payload import New, NewFile, NewStored
from synnax.secret.types_gen import (
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Secret,
    Where,
)
from synnax.util import one
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    secrets: list[New]


class _CreateResponse(BaseModel):
    secrets: list[Secret]


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, value: str) -> Secret: ...

    @overload
    def create(self, *, name: str, path: str) -> Secret: ...

    @overload
    def create(self, secrets: New) -> Secret: ...

    @overload
    def create(self, secrets: list[New]) -> list[Secret]: ...

    def create(
        self,
        secrets: New | list[New] | None = None,
        *,
        name: str | None = None,
        value: str | None = None,
        path: str | None = None,
    ) -> Secret | list[Secret]:
        """Creates secrets, or replaces the secrets whose keys exist.

        :param secrets: A secret or a list of them. When omitted, a stored secret is
            created from ``name`` and ``value``, or a file secret from ``name`` and
            ``path``.
        :param name: The name of the secret when ``secrets`` is omitted.
        :param value: The value of a stored secret. No client can read it back.
        :param path: The path of a file secret on the computer that uses it.
        :returns: The saved secret, or a list when ``secrets`` is a list. Values are
            never returned.
        :raises ValidationError: If ``secrets`` is omitted and the keyword arguments do
            not give a name and exactly one of a value or a path.
        :raises PathError: If a secret is not valid, such as a stored secret without a
            value.
        :raises AccessDenied: If the caller cannot create secrets, or cannot update a
            secret that the call replaces.
        """
        is_single = not isinstance(secrets, list)
        if secrets is None:
            if name is not None and value is not None and path is None:
                secrets = NewStored(name=name, value=value)
            elif name is not None and path is not None and value is None:
                secrets = NewFile(name=name, path=path)
            else:
                raise ValidationError(
                    "A secret needs a name and exactly one of a value or a path."
                )
        req = _CreateRequest(secrets=normalize(secrets))
        res = self._client.send("/secret/create", req, _CreateResponse)
        return res.secrets[0] if is_single else res.secrets

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes secrets. Unknown keys are ignored.

        :param keys: The keys of the secrets to delete.
        :raises ValidationError: If a resource uses one of the secrets.
        :raises AccessDenied: If the caller cannot delete one of the secrets.
        """
        self._client.send("/secret/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a secret.

        :param key: The key of the secret to rename.
        :param name: The new name for the secret.
        :raises PathError: If the name is empty.
        :raises NotFoundError: If no secret has the given key.
        :raises AccessDenied: If the caller cannot update the secret.
        """
        self._client.send("/secret/rename", _RenameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, *, key: Key, include_uses: bool = False) -> Secret: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_uses: bool = False,
    ) -> list[Secret]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_uses: bool = False,
    ) -> Secret | list[Secret]:
        """Retrieves secrets, without their values.

        :param key: The key of the one secret to retrieve.
        :param where: A filter that selects the secrets, combined with ``keys``.
        :param keys: The keys of the secrets to retrieve. A key that no secret has is
            left out.
        :param search_term: Keeps the secrets whose names match, closest match first.
        :param limit: The most secrets to return.
        :param offset: The number of matching secrets to skip.
        :param include_uses: Whether to fill each secret's uses with the ontology IDs of
            the resources that use it and that the caller can read. Secrets come back
            with no uses when false.
        :returns: The secret when ``key`` is given, otherwise a list.
        :raises NotFoundError: If ``key`` is given and no secret has it.
        :raises AccessDenied: If the caller cannot retrieve one of the secrets.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        res = self._client.send(
            "/secret/retrieve",
            RetrieveRequest(
                where=filter.merge(where, flat),
                search_term=search_term,
                limit=limit,
                offset=offset,
                include_uses=include_uses,
            ),
            RetrieveResponse,
        )
        if is_single:
            return one("Secret", key, res.secrets)
        return res.secrets
