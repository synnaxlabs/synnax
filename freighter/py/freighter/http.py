from __future__ import annotations

import dataclasses
from typing import Generic, Type

from .encoder import EncoderDecoder
from .transport import RS, RQ, PayloadFactory, PayloadFactoryFunc
from urllib3 import PoolManager, HTTPResponse
from urllib.parse import urlencode
from .errors import ErrorPayload, decode
from .endpoint import Endpoint
from urllib3.exceptions import HTTPError

http = PoolManager()

_ERROR_ENCODING_HEADER_KEY = "Error-Encoding"
_ERROR_ENCODING_HEADER_VALUE = "freighter"
_CONTENT_TYPE_HEADER_KEY = "Content-Type"


class Client:
    endpoint: Endpoint
    encoder_decoder: EncoderDecoder

    def __init__(
            self, endpoint: Endpoint, encoder_decoder: EncoderDecoder
    ):
        self.endpoint = endpoint
        self.encoder_decoder = encoder_decoder

    def get(
            self, request_type: Type[RQ], response_factory: PayloadFactoryFunc[RS]
    ) -> GETClient[RQ, RS]:
        return GETClient(self.endpoint, self.encoder_decoder, response_factory)

    def post(
            self, request_type: Type[RQ], response_factory: PayloadFactoryFunc[RS]
    ) -> POSTClient[RQ, RS]:
        return POSTClient(self.endpoint, self.encoder_decoder, response_factory)


class _Core(Generic[RQ, RS]):
    endpoint: Endpoint
    encoder_decoder: EncoderDecoder
    response_factory: PayloadFactory[RS]

    def __init__(
            self,
            endpoint: Endpoint,
            encoder_decoder: EncoderDecoder,
            response_factory: PayloadFactoryFunc[RS]
    ):
        self.endpoint = endpoint.child("", "http")
        self.encoder_decoder = encoder_decoder
        self.response_factory = PayloadFactory[RS](response_factory)

    @property
    def headers(self) -> dict[str, str]:
        return {
            _CONTENT_TYPE_HEADER_KEY: self.encoder_decoder.content_type(),
            _ERROR_ENCODING_HEADER_KEY: _ERROR_ENCODING_HEADER_VALUE,
        }


class GETClient(_Core[RQ, RS]):
    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        query_args = build_query_string(req)
        url = self.endpoint.path(target) + "?" + query_args
        http_res: HTTPResponse
        try:
            http_res = http.request(method="GET", url=url, headers=self.headers)
        except HTTPError as e:
            return None, e

        if http_res.status != 200:
            err = ErrorPayload(None, None)
            self.encoder_decoder.decode(http_res.data, err)
            return None, decode(err)

        res = self.response_factory()
        self.encoder_decoder.decode(http_res.data, res)
        return res, None

    def post_client(self) -> POSTClient:
        return POSTClient(self.endpoint, self.encoder_decoder, self.response_factory)


class POSTClient(_Core[RQ, RS]):
    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        url = self.endpoint.path(target)
        http_res: HTTPResponse
        try:
            http_res = http.request(
                method="POST",
                url=url,
                headers=self.headers,
                body=self.encoder_decoder.encode(req),
            )
        except HTTPError as e:
            return None, e

        if http_res.status != 200 and http_res.status != 201:
            err = ErrorPayload(None, None)
            self.encoder_decoder.decode(http_res.data, err)
            return None, decode(err)

        res = self.response_factory()
        self.encoder_decoder.decode(http_res.data, res)
        return res, None

    def get_client(self) -> GETClient:
        return GETClient(self.endpoint, self.encoder_decoder, self.response_factory)


def build_query_string(req: RQ) -> str:
    raw_dct = dataclasses.asdict(req)
    parsed_dct = dict()
    for key, val in raw_dct.items():
        if val is not None:
            parsed_dct[key] = val
        if type(val) is list:
            parsed_dct[key] = ",".join(val)
    return urlencode(parsed_dct)
