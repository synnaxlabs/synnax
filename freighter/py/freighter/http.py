from __future__ import annotations

from typing import Generic, Type
from urllib.parse import urlencode

from urllib3 import HTTPResponse, PoolManager
from urllib3.exceptions import HTTPError

from .encoder import EncoderDecoder
from .exceptions import ExceptionPayload, decode_exception
from .transport import RQ, RS
from .url import URL

http = PoolManager()


class HTTPClient:
    """HTTPClient provides a POST and GET implementation of the UnaryClient protocol.

    :param endpoint: The base URL for the client.
    :param encoder_decoder: The encoder/decoder to use for the client.
    """

    endpoint: URL
    encoder_decoder: EncoderDecoder

    def __init__(self, endpoint: URL, encoder_decoder: EncoderDecoder):
        self.endpoint = endpoint
        self.encoder_decoder = encoder_decoder

    def client_get(self, req_t: Type[RQ], res_t: Type[RS]) -> GETClient[RQ, RS]:
        """Creates a GET client for the given request and response types.

        :param req_t: The request type.
        :param res_t: The response type.
        :return: A GET client for the given request and response types.
        """
        return GETClient(self.endpoint, self.encoder_decoder, res_t)

    def client_post(self, req_t: Type[RQ], res_t: Type[RS]) -> POSTClient[RQ, RS]:
        """Creates a POST client for the given request and response types.

        :param req_t: The request type.
        :param res_t: The response type.
        :return: A POST client for the given request and response types.
        """
        return POSTClient(self.endpoint, self.encoder_decoder, res_t)


class _Core(Generic[RQ, RS]):
    _ERROR_ENCODING_HEADER_KEY = "Error-Encoding"
    _ERROR_ENCODING_HEADER_VALUE = "freighter"
    _CONTENT_TYPE_HEADER_KEY = "Content-Type"

    endpoint: URL
    encoder_decoder: EncoderDecoder
    res_t: Type[RS]

    def __init__(
        self,
        endpoint: URL,
        encoder_decoder: EncoderDecoder,
        res_t: Type[RS],
    ):
        self.endpoint = endpoint.replace(protocol="http")
        self.encoder_decoder = encoder_decoder
        self.res_t = res_t

    @property
    def headers(self) -> dict[str, str]:
        return {
            self._CONTENT_TYPE_HEADER_KEY: self.encoder_decoder.content_type(),
            self._ERROR_ENCODING_HEADER_KEY: self._ERROR_ENCODING_HEADER_VALUE,
        }

    def request(
        self, method: str, url: str, request: RQ | None = None
    ) -> tuple[RS | None, Exception | None]:

        data = None
        if request is not None:
            data = self.encoder_decoder.encode(request)

        http_res: HTTPResponse
        try:
            http_res = http.request(
                method=method, url=url, headers=self.headers, body=data
            )
        except HTTPError as e:
            return None, e

        if http_res.status != 200 and http_res.status != 201:
            err = self.encoder_decoder.decode(http_res.data, ExceptionPayload)
            return None, decode_exception(err)

        if http_res.data is None:
            return None, None

        return self.encoder_decoder.decode(http_res.data, self.res_t), None


class GETClient(_Core[RQ, RS]):
    """Implementation of the UnaryClient protocol backed by HTTP GET requests.

    :param endpoint: The base URL for the client.
    :param encoder_decoder: The encoder/decoder to use for the client.
    :param res_t: The response type.
    """

    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        """Implements the UnaryClient protocol."""
        return self.request("GET", self._build_url(target, req))

    def client_post(self) -> POSTClient:
        """Creates a POST client for the same endpoint and request and response types.

        :return: A POST client for the same endpoint and request and response types.
        """
        return POSTClient(self.endpoint, self.encoder_decoder, self.res_t)

    def _build_url(self, target: str, req: RQ):
        base = self.endpoint.child(target)
        return base.stringify() + "?" + self._build_query_string(req)

    @staticmethod
    def _build_query_string(req: RQ) -> str:
        raw_dct = req.dict()
        parsed_dct = dict()
        for key, val in raw_dct.items():
            if val is not None:
                parsed_dct[key] = val
            if type(val) is list:
                parsed_dct[key] = ",".join(val)
        return urlencode(parsed_dct)


class POSTClient(_Core[RQ, RS]):
    """Implementation of the UnaryClient protocol backed by HTTP POST requests.

    :param endpoint: The base URL for the client.
    :param encoder_decoder: The encoder/decoder to use for the client.
    :param res_t: The response type.
    """

    def send(self, target: str, req: RQ) -> tuple[RS | None, Exception | None]:
        """Implements the UnaryClient protocol."""
        return self.request("POST", self.endpoint.child(target).stringify(), req)

    def client_get(self) -> GETClient:
        """Creates a GET client for the same endpoint and request and response types.

        :return: A GET client for the same endpoint and request and response types.
        """
        return GETClient(self.endpoint, self.encoder_decoder, self.res_t)
