from __future__ import annotations

import dataclasses
from typing import Generic, Type
from urllib.parse import urlencode

from urllib3 import HTTPResponse, PoolManager
from urllib3.exceptions import HTTPError

from .encoder import EncoderDecoder
from .errors import ErrorPayload, decode
from .transport import RQ, RS
from .url import URL

http = PoolManager()

_ERROR_ENCODING_HEADER_KEY = "Error-Encoding"
_ERROR_ENCODING_HEADER_VALUE = "freighter"
_CONTENT_TYPE_HEADER_KEY = "Content-Type"


class HTTPClient:
    endpoint: URL
    encoder_decoder: EncoderDecoder

    def __init__(self, endpoint: URL, encoder_decoder: EncoderDecoder):
        self.endpoint = endpoint
        self.encoder_decoder = encoder_decoder

    def get(self, request_type: Type[RQ], response_type: Type[RS]) -> GETClient[RQ, RS]:
        return GETClient(self.endpoint, self.encoder_decoder, response_type)

    def post(self, request_type: Type[RQ], rs_t: Type[RS]) -> POSTClient[RQ, RS]:
        return POSTClient(self.endpoint, self.encoder_decoder, rs_t)


class _Core(Generic[RQ, RS]):
    endpoint: URL
    encoder_decoder: EncoderDecoder
    response_factory: Type[RS]

    def __init__(
        self,
        endpoint: URL,
        encoder_decoder: EncoderDecoder,
        res_t: Type[RS],
    ):
        self.endpoint = endpoint.replace(protocol="http")
        self.encoder_decoder = encoder_decoder
        self.response_factory = res_t

    @property
    def headers(self) -> dict[str, str]:
        return {
            _CONTENT_TYPE_HEADER_KEY: self.encoder_decoder.content_type(),
            _ERROR_ENCODING_HEADER_KEY: _ERROR_ENCODING_HEADER_VALUE,
        }


class GETClient(_Core[RQ, RS]):
    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        query_args = build_query_string(req)
        url = self.endpoint.child(target).stringify() + "?" + query_args
        http_res: HTTPResponse
        try:
            http_res = http.request(method="GET", url=url, headers=self.headers)
        except HTTPError as e:
            return None, e

        if http_res.status != 200:
            err = ErrorPayload(None, None)
            self.encoder_decoder.decode(http_res.data, err)
            return None, decode(err)

        res = self.response_factory.new()
        self.encoder_decoder.decode(http_res.data, res)
        return res, None

    def post_client(self) -> POSTClient:
        return POSTClient(self.endpoint, self.encoder_decoder, self.response_factory)


class POSTClient(_Core[RQ, RS]):
    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        url = self.endpoint.child(target).stringify()
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

        res = self.response_factory.new()
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
