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
from synnax.certificate.payload import New
from synnax.certificate.types_gen import (
    Certificate,
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Where,
)
from synnax.util import one
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    certificates: list[New]


class _CreateResponse(BaseModel):
    certificates: list[Certificate]


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
    def create(self, certificates: New) -> Certificate: ...

    @overload
    def create(self, certificates: list[New]) -> list[Certificate]: ...

    def create(self, certificates: New | list[New]) -> Certificate | list[Certificate]:
        """Creates certificates, or replaces the certificates whose keys exist.

        :param certificates: The certificates to create.
        :returns: The saved certificates.
        :raises PathError: If a certificate has no name, a PEM does not parse, or a
            private key does not match its certificate.
        :raises AccessDenied: If the caller may not create certificates, create secrets
            for private keys, or update a certificate it replaces.
        """
        is_single = not isinstance(certificates, list)
        req = _CreateRequest(certificates=normalize(certificates))
        res = self._client.send("/certificate/create", req, _CreateResponse)
        return res.certificates[0] if is_single else res.certificates

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes certificates and the secrets that hold their private keys.

        :param keys: The keys of the certificates to delete.
        :raises AccessDenied: If the caller may not delete a certificate.
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
        :raises AccessDenied: If the caller may not update the certificate.
        """
        self._client.send(
            "/certificate/rename", _RenameRequest(key=key, name=name), Empty
        )

    @overload
    def retrieve(self, *, key: Key) -> Certificate: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Certificate]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Certificate | list[Certificate]:
        """Retrieves certificates.

        :param key: The key of the one certificate to retrieve.
        :param where: A filter that selects the certificates, combined with ``keys``.
        :param keys: The keys of the certificates to retrieve. A key that no
            certificate has is left out.
        :param search_term: Keeps the certificates whose names match, closest match
            first.
        :param limit: The most certificates to return.
        :param offset: The number of matching certificates to skip.
        :returns: The certificate when ``key`` is given, otherwise a list.
        :raises NotFoundError: If key is given and no certificate has it.
        :raises AccessDenied: If the caller may not retrieve a matching certificate.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        res = self._client.send(
            "/certificate/retrieve",
            RetrieveRequest(
                where=filter.merge(where, flat),
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        if is_single:
            return one("Certificate", key, res.certificates)
        return res.certificates
