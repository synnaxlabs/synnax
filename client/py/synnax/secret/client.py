#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.secret.payload import New
from synnax.secret.types_gen import Key, Secret
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


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(BaseModel):
    secrets: list[Secret] = Field(default_factory=list)


class Client:
    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, secrets: New) -> Secret: ...

    @overload
    def create(self, secrets: list[New]) -> list[Secret]: ...

    def create(self, secrets: New | list[New]) -> Secret | list[Secret]:
        """Creates secrets, or replaces the secrets whose keys exist.

        :param secrets: The secrets to create.
        :returns: The saved secrets, without their values.
        """
        is_single = not isinstance(secrets, list)
        req = _CreateRequest(secrets=normalize(secrets))
        res = self._client.send("/secret/create", req, _CreateResponse)
        return res.secrets[0] if is_single else res.secrets

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes secrets.

        :param keys: The keys of the secrets to delete.
        :raises ValidationError: If a device or certificate uses one of the secrets.
        """
        self._client.send("/secret/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a secret.

        :param key: The key of the secret to rename.
        :param name: The new name for the secret.
        :raises NotFoundError: If no secret has the given key.
        """
        self._client.send("/secret/rename", _RenameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, *, key: Key) -> Secret: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[Secret]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> Secret | list[Secret]:
        """Retrieves secrets, without their values.

        :raises NotFoundError: If key is given and no secret has it.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/secret/retrieve",
            _RetrieveRequest(
                keys=keys, search_term=search_term, limit=limit, offset=offset
            ),
            _RetrieveResponse,
        )
        if not is_single:
            return res.secrets
        if len(res.secrets) == 0:
            raise NotFoundError("Secret not found")
        return res.secrets[0]
