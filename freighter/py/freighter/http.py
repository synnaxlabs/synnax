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
from urllib.parse import urlencode

import urllib3
from urllib3 import HTTPResponse, PoolManager
from urllib3.exceptions import HTTPError, MaxRetryError

from .encoder import EncoderDecoder
from .exceptions import ExceptionPayload, decode_exception, Unreachable
from .transport import RQ, RS
from .url import URL
from .transport import MiddlewareCollector
from .metadata import MetaData


class HTTPClientPool(MiddlewareCollector):
    """HTTPClientFactory provides a factory for creating POST and GET implementation of
    the UnaryClient protocol.
    """

    pool: PoolManager
    endpoint: URL
    encoder_decoder: EncoderDecoder
    secure: bool

    def __init__(
        self, url: URL, encoder_decoder: EncoderDecoder, secure: bool = False, **kwargs
    ):
        """
        :param url: The base URL for the client.
        :param encoder_decoder: The encoder/decoder to use for the client.
        :param secure: Whether to use HTTPS.
        """
        super().__init__()
        self.endpoint = url
        self.encoder_decoder = encoder_decoder
        self.secure = secure
        self.pool = PoolManager(cert_reqs="CERT_NONE", **kwargs)
        urllib3.disable_warnings()

    def get_client(self) -> GETClient:
        """Creates a GET client for the given request and response types.
        :returns: A GET client for the given request and response types.
        """
        gc = GETClient(
            self.endpoint, self.encoder_decoder, self.pool, secure=self.secure
        )
        gc.use(*self._middleware)
        return gc

    def post_client(self) -> POSTClient:
        """:returns: A POST client for the given request and response types."""
        pc = POSTClient(
            self.endpoint, self.encoder_decoder, self.pool, secure=self.secure
        )
        pc.use(*self._middleware)
        return pc


class _Core(MiddlewareCollector):
    _ERROR_ENCODING_HEADER_KEY = "Error-Encoding"
    _ERROR_ENCODING_HEADER_VALUE = "freighter"
    _CONTENT_TYPE_HEADER_KEY = "Content-Type"

    pool: PoolManager
    endpoint: URL
    encoder_decoder: EncoderDecoder
    res: RS | None

    def __init__(
        self,
        endpoint: URL,
        encoder_decoder: EncoderDecoder,
        pool: PoolManager,
        secure: bool = False,
    ):
        super().__init__()
        self.endpoint = endpoint.replace(protocol="https" if secure else "http")
        self.encoder_decoder = encoder_decoder
        self.res = None
        self.pool = pool

    @property
    def headers(self) -> dict[str, str]:
        return {
            self._CONTENT_TYPE_HEADER_KEY: self.encoder_decoder.content_type(),
            self._ERROR_ENCODING_HEADER_KEY: self._ERROR_ENCODING_HEADER_VALUE,
        }

    def request(
        self,
        method: str,
        url: str,
        request: RQ | None = None,
        res_t: Type[RS] | None = None,
    ) -> tuple[RS | None, Exception | None]:
        in_meta_data = MetaData(url, self.endpoint.protocol)

        def finalizer(md: MetaData) -> tuple[MetaData, Exception | None]:
            out_meta_data = MetaData(url, self.endpoint.protocol)
            data = None
            if request is not None:
                data = self.encoder_decoder.encode(request)

            head = {**self.headers, **md.params}

            http_res: HTTPResponse
            try:
                http_res = self.pool.request(
                    method=method, url=url, headers=head, body=data
                )
            except MaxRetryError as e:
                return out_meta_data, Unreachable(url, e)
            except HTTPError as e:
                return out_meta_data, e

            out_meta_data.params = http_res.headers

            if http_res.status < 200 or http_res.status >= 300:
                err = self.encoder_decoder.decode(http_res.data, ExceptionPayload)
                return out_meta_data, decode_exception(err)

            if http_res.data is None:
                return out_meta_data, None

            self.res = self.encoder_decoder.decode(http_res.data, res_t)
            return out_meta_data, None

        _, exc = self.exec(in_meta_data, finalizer)
        return self.res, exc


class GETClient(_Core):
    """Implementation of the UnaryClient protocol backed by HTTP GET requests. It should
    not be instantiated directly, but through the HTTPClientFactory.
    """

    def send(
        self, target: str, req: RQ, res_t: Type[RS]
    ) -> tuple[RS | None, Exception | None]:
        """Implements the UnaryClient protocol."""
        return self.request("GET", self._build_url(target, req), None, res_t)

    def client_post(self) -> POSTClient:
        """Creates a POST client for the same endpoint and request and response types.

        :return: A POST client for the same endpoint and request and response types.
        """
        return POSTClient(self.endpoint, self.encoder_decoder)

    def _build_url(self, target: str, req: RQ):
        base = self.endpoint.child(target)
        return base.stringify() + "?" + self._build_query_string(req)

    @staticmethod
    def _build_query_string(req: RQ) -> str:
        raw_dct = req.dict()
        parsed_dct = dict()
        for key, val in raw_dct.items():
            if type(val) is list:
                if len(val) > 0:
                    parsed_dct[key] = ",".join(val)
            elif val is not None:
                parsed_dct[key] = val
        return urlencode(parsed_dct)


class POSTClient(_Core):
    """Implementation of the UnaryClient protocol backed by HTTP POST requests. it should
    not be instantiated directly, but through the HTTPClientFactory.
    """

    def send(
        self, target: str, req: RQ, res_t: Type[RS]
    ) -> tuple[RS | None, Exception | None]:
        """Implements the UnaryClient protocol."""
        return self.request("POST", self.endpoint.child(target).stringify(), req, res_t)

    def client_get(self) -> GETClient:
        """Creates a GET client for the same endpoint and request and response types.

        :return: A GET client for the same endpoint and request and response types.
        """
        return GETClient(self.endpoint, self.encoder_decoder)
