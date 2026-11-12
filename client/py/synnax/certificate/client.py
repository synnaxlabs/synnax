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
from synnax.certificate import actions_gen
from synnax.certificate.payload import New
from synnax.certificate.types_gen import (
    Certificate,
    Cursor,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _SetRequest(BaseModel):
    certificates: list[New]


class _SetResponse(BaseModel):
    certificates: list[Certificate]


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def set(self, certificates: New) -> Certificate: ...

    @overload
    def set(self, certificates: list[New]) -> list[Certificate]: ...

    def set(self, certificates: New | list[New]) -> Certificate | list[Certificate]:
        """Creates certificates, or replaces the certificates whose keys exist.

        :param certificates: The certificates to create or replace.
        :returns: The saved certificates.
        :raises PathError: If a certificate has no name, a PEM does not parse, or a
            private key does not match its certificate.
        :raises AccessDenied: If the caller may not create certificates, create secrets
            for private keys, or update a certificate it replaces and the secret of its
            private key.
        """
        is_single = not isinstance(certificates, list)
        req = _SetRequest(certificates=normalize(certificates))
        res = self._client.send("/certificate/set", req, _SetResponse)
        return res.certificates[0] if is_single else res.certificates

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes certificates and the secrets that hold their private keys.

        :param keys: The keys of the certificates to delete.
        :raises AccessDenied: If the caller may not delete a certificate or the secret
            of its private key.
        """
        self._client.send(
            "/certificate/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def rename(self, key: Key, name: str) -> None:
        """Renames a certificate.

        :param key: The key of the certificate to rename.
        :param name: The new name for the certificate.
        :raises PathError: If the name is empty.
        :raises NotFoundError: If no certificate has the given key.
        :raises AccessDenied: If the caller may not update the certificate, or the
            secret of its private key, which takes the new name.
        """
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/certificate/dispatch", req, Empty)

    @overload
    def retrieve(self, key: Key) -> Certificate: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Certificate: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Certificate: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Certificate: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Certificate]: ...

    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Certificate | list[Certificate]:
        """Retrieves certificates. With no arguments, retrieves every certificate.

        Each filter field takes a value, a list of values, or an operator, and a
        certificate must match every field. A plain ``key``, by position or by keyword,
        or a plain ``name`` returns one certificate.

        :param key: Keeps certificates with this key.
        :param name: Keeps certificates with this name.
        :param and_: Keeps certificates that match every filter in the list.
        :param or_: Keeps certificates that match any filter in the list.
        :param not_: Keeps certificates that do not match the filter.
        :param where: A filter tree that the certificates must also match.
        :param keys: Keeps certificates with one of these keys.
        :param search_term: Keeps the certificates whose names match, closest match
            first.
        :param limit: The most certificates to return.
        :param order: The order of the certificates. Defaults to key order.
        :param after: Retrieves the certificates past this position in the order: the
            key of the last certificate of the page before, and its value of the order
            field.
        :param one: Returns one certificate for any filter.
        :returns: One certificate for a plain ``key`` or ``name`` or when ``one`` is
            set, otherwise a list of certificates. The list leaves out keys that no
            certificate has.
        :raises NotFoundError: If one certificate is wanted and none matches.
        :raises MultipleFoundError: If one certificate is wanted and more than one
            matches.
        :raises ValidationError: If a filter field is None, or if one certificate is
            wanted beside a limit or a cursor.
        :raises AccessDenied: If the caller may not retrieve a matching certificate.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/certificate/retrieve", req, RetrieveResponse)
        return (
            sole("certificate", req, res.certificates) if single else res.certificates
        )
