#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Type

import urllib3
from urllib3 import PoolManager
from urllib3.exceptions import HTTPError, MaxRetryError
from urllib3.response import BaseHTTPResponse

from freighter.context import Context, Role
from freighter.encoder import EncoderDecoder
from freighter.exceptions import ExceptionPayload, Unreachable, decode_exception
from freighter.transport import RQ, RS, MiddlewareCollector
from freighter.unary import UnaryClient
from freighter.url import URL


class HTTPClient(MiddlewareCollector):
    """HTTPClientFactory provides a factory for creating POST and GET implementation of
    the UnaryClient protocol.
    """

    __ERROR_ENCODING_HEADER_KEY = "Error-Encoding"
    __ERROR_ENCODING_HEADER_VALUE = "freighter"
    __CONTENT_TYPE_HEADER_KEY = "Content-Type"
    __pool: PoolManager
    __endpoint: URL
    __encoder_decoder: EncoderDecoder
    __secure: bool

    def __init__(
        self,
        url: URL,
        encoder_decoder: EncoderDecoder,
        secure: bool = False,
        **kwargs,
    ):
        """
        :param url: The base URL for the client.
        :param encoder_decoder: The encoder/decoder to use for the client.
        :param secure: Whether to use HTTPS.
        """
        super().__init__()
        self.__endpoint = url
        self.__endpoint.protocol = "https" if secure else "http"
        self.__encoder_decoder = encoder_decoder
        self.__secure = secure
        self.__pool = PoolManager(cert_reqs="CERT_NONE", **kwargs)
        urllib3.disable_warnings()

    def __(self) -> UnaryClient:
        return self

    def send(
        self, target: str, req: RQ, res_t: Type[RS]
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """Implements the UnaryClient protocol."""
        return self.request(
            "POST",
            self.__endpoint.child(target).stringify(),
            "client",
            req,
            res_t,
        )

    @property
    def __headers(self) -> dict[str, str]:
        return {
            self.__CONTENT_TYPE_HEADER_KEY: self.__encoder_decoder.content_type(),
            self.__ERROR_ENCODING_HEADER_KEY: self.__ERROR_ENCODING_HEADER_VALUE,
        }

    def request(
        self,
        method: str,
        url: str,
        role: Role,
        request: RQ | None = None,
        res_t: Type[RS] | None = None,
    ) -> tuple[RS, None] | tuple[None, Exception]:
        in_ctx = Context(url, self.__endpoint.protocol, role)

        res: RS | None = None

        def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            nonlocal res
            out_meta_data = Context(url, self.__endpoint.protocol, role)
            data = None
            if request is not None:
                data = self.__encoder_decoder.encode(request)

            head = {**self.__headers, **ctx.params}

            http_res: BaseHTTPResponse
            try:
                http_res = self.__pool.request(
                    method=method, url=url, headers=head, body=data
                )
            except MaxRetryError as e:
                return out_meta_data, Unreachable(url, e.url)
            except HTTPError as e:
                return out_meta_data, e

            out_meta_data.params = http_res.headers

            if http_res.status < 200 or http_res.status >= 300:
                err = self.__encoder_decoder.decode(http_res.data, ExceptionPayload)
                return out_meta_data, decode_exception(err)

            if http_res.data is None:
                return out_meta_data, None

            res = self.__encoder_decoder.decode(http_res.data, res_t)
            return out_meta_data, None

        _, exc = self.exec(in_ctx, finalizer)
        return res, exc
