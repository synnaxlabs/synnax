import dataclasses
from typing import Generic

from .encoder import EncoderDecoder
from .transport import RS, RQ, Payload, PayloadFactory, PayloadFactoryFunc
from urllib3 import PoolManager
from urllib.parse import urlencode, urljoin
import urllib3
from .errors import ErrorPayload, decode
from .endpoint import Endpoint

http = PoolManager()

ERROR_ENCODING_HEADER = "freighter"


class _HTTPClient(Generic[RS]):
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


class GETClient(Generic[RQ, RS], _HTTPClient):
    async def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        query_args = build_query_string(req)
        url = self.endpoint.build(target) + "?" + query_args
        http_res: urllib3.HTTPResponse
        try:
            http_res = http.request(method="GET", url=url, headers=self.headers)
        except urllib3.exceptions.HTTPError as e:
            return e

        if http_res.status != 200:
            err = ErrorPayload(None, None)
            self.encoder_decoder.decode(http_res.data, err)
            return None, decode(err)

        self.encoder_decoder.decode(http_res.data, self.response_factory())
        return None, None


class POSTClient(Generic[RQ, RS], _HTTPClient):
    async def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        url = self.endpoint.build(target)
        http_res: urllib3.HTTPResponse
        try:
            http_res = http.request(
                method="POST",
                url=url,
                headers=self.headers,
                body=self.encoder_decoder.encode(req),
            )
        except urllib3.exceptions.HTTPError as e:
            return e

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
