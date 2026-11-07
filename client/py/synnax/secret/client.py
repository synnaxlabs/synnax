#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

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
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
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
    def retrieve(self, key: Key) -> Secret: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Secret: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Secret: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        offset: int = ...,
        one: Literal[False] = ...,
    ) -> list[Secret]: ...

    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        one: bool = False,
        **fields: Any,
    ) -> Secret | list[Secret]:
        """Retrieves secrets, without their values. With no arguments, retrieves every
        secret.

        Each filter field takes a value, a list of values, or an operator, and a secret
        must match every field. A plain ``key``, by position or by keyword, returns one
        secret.

        :param key: Keeps secrets with this key.
        :param and_: Keeps secrets that match every filter in the list.
        :param or_: Keeps secrets that match any filter in the list.
        :param not_: Keeps secrets that do not match the filter.
        :param where: A filter tree that the secrets must also match.
        :param keys: Keeps secrets with one of these keys.
        :param search_term: Keeps the secrets whose names match, closest match first.
        :param limit: The most secrets to return.
        :param offset: The number of matching secrets to skip.
        :param one: Returns one secret for any filter.
        :returns: One secret for a plain ``key`` or when ``one`` is set, otherwise a
            list of secrets. The list leaves out keys that no secret has.
        :raises NotFoundError: If one secret is wanted and none matches.
        :raises MultipleFoundError: If one secret is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one secret is wanted
            beside a limit or an offset.
        :raises AccessDenied: If the caller cannot retrieve one of the secrets.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit, offset)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            offset=offset,
        )
        res = self._client.send("/secret/retrieve", req, RetrieveResponse)
        return sole("Secret", fields, res.secrets) if single else res.secrets
