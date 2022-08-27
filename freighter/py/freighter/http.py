import dataclasses
from typing import Generic

from .encoder import EncoderDecoder
from .transport import RS, RQ, PayloadFactory, PayloadFactoryFunc
from urllib3 import PoolManager, HTTPResponse
from urllib.parse import urlencode
from .errors import ErrorPayload, decode
from .endpoint import Endpoint
from urllib3.exceptions import HTTPError

http = PoolManager()

ERROR_ENCODING_HEADER = "freighter"


class _HTTPClient(Generic[RQ, RS]):
    endpoint: Endpoint
    encoder_decoder: EncoderDecoder
    response_factory: PayloadFactory[RS]

    def __init__(
            self,
            endpoint: Endpoint,
            encoder_decoder: EncoderDecoder,
            response_factory: PayloadFactoryFunc[RS]
    ):
        self.endpoint = endpoint
        self.encoder_decoder = encoder_decoder
        self.response_factory = PayloadFactory[RS](response_factory)

    @property
    def headers(self) -> dict[str, str]:
        return {
            "Content-Type": self.encoder_decoder.content_type(),
            "Error-Encoding": "freighter",
        }


class GETClient(_HTTPClient):
    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        query_args = build_query_string(req)
        url = self.endpoint.build(target) + "?" + query_args
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


class POSTClient(_HTTPClient):
    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        url = self.endpoint.build(target)
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


def build_query_string(req: RQ) -> str:
    raw_dct = dataclasses.asdict(req)
    parsed_dct = dict()
    for key, val in raw_dct.items():
        if val is not None:
            parsed_dct[key] = val
        if type(val) is list:
            parsed_dct[key] = ",".join(val)
    return urlencode(parsed_dct)
