#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import os
import pathlib
from collections.abc import Iterable, Sequence
from typing import IO, Any, Protocol

import urllib3
from pydantic import BaseModel
from urllib3 import PoolManager
from urllib3.exceptions import MaxRetryError
from urllib3.response import BaseHTTPResponse

from freighter.codec import Codec
from freighter.context import Context
from freighter.exceptions import Unreachable
from freighter.transport import RQ, RS, FilePath, MiddlewareCollector
from freighter.url import URL
from x.exceptions import ExceptionPayload, decode_exception

_CONTENT_TYPE_HEADER_KEY = "Content-Type"


class FileCodec(Codec, Protocol):
    """A Codec that also has an on-disk file representation.

    HTTPClient uses file_extension to negotiate a content type from a file path during
    upload and download. Codecs that only exist on a wire (e.g., a WebSocket framer
    codec) need only satisfy Codec.
    """

    def file_extension(self) -> str:
        """:returns: the file extension (without leading dot) associated with the
        Codec's on-disk format, e.g. "json" or "msgpack".
        """
        ...


class HTTPClient(MiddlewareCollector):
    """
    HTTPClient is a urllib3-backed unary transport implementing UnaryClient,
    UploadClient, and DownloadClient:

    - send: typed request, typed response, both via the configured encoder/decoders.
    - upload: a file path as the request body, with a typed response. Bytes are streamed
      from disk via chunked transfer; the Content-Type is inferred from the path's
      extension.
    - download: a typed request, with the response streamed directly into a destination
      file path; the Accept header is derived from the destination extension.
    """

    _pool: PoolManager
    _endpoint: URL
    _encoder: FileCodec
    _decoders: tuple[FileCodec, ...]
    _decoders_by_content_type: dict[str, FileCodec]
    _codecs_by_extension: dict[str, FileCodec]
    _accept_header: str

    def __init__(
        self,
        url: URL,
        encoder: FileCodec,
        decoders: Sequence[FileCodec],
        secure: bool = False,
        **kwargs: Any,
    ) -> None:
        """
        :param url: The base URL for the client.
        :param encoder: The codec used to encode outgoing typed requests.
        :param decoders: The codecs the client is willing to decode responses from. Sent
            as the Accept header. Must not be empty.
        :param secure: Whether to use HTTPS.
        """
        if len(decoders) == 0:
            raise ValueError("HTTPClient requires at least one response decoder")
        super().__init__()
        self._endpoint = url
        self._endpoint.protocol = "https" if secure else "http"
        self._encoder = encoder
        self._decoders = tuple(decoders)
        self._decoders_by_content_type = {d.content_type(): d for d in self._decoders}
        self._codecs_by_extension = {
            c.file_extension(): c for c in (encoder, *self._decoders)
        }
        self._accept_header = ", ".join(d.content_type() for d in self._decoders)
        self._pool = PoolManager(cert_reqs="CERT_NONE", **kwargs)
        urllib3.disable_warnings()

    def send(self, target: str, req: RQ, res_t: type[RS]) -> RS:
        """Implements the UnaryClient protocol — typed request, typed response."""
        return self._typed_response_request(
            target=target,
            body=self._encoder.encode(req),
            content_type=self._encoder.content_type(),
            res_t=res_t,
        )

    def upload(self, target: str, req: FilePath, res_t: type[RS]) -> RS:
        """
        Streams the file at req to target and decodes the response into res_t. urllib3
        uses chunked transfer encoding so the body never has to fit in memory; the
        Content-Type is inferred from the file extension via the client's registered
        codecs.
        """
        codec = self._codec_for_path(req)
        if isinstance(codec, Exception):
            raise codec
        with open(req, "rb") as f:
            return self._typed_response_request(
                target=target,
                body=f,
                content_type=codec.content_type(),
                res_t=res_t,
            )

    def download(self, target: str, req: BaseModel, dest: FilePath) -> None:
        """
        Sends req to target and streams the response body directly into dest, without
        buffering the full body in memory. The Accept header is derived from the dest
        extension via the client's registered codecs (e.g., a .json destination requests
        application/json), so the on-disk format and the negotiated wire format are
        guaranteed to match.
        """
        dest_codec = self._codec_for_path(dest)
        if isinstance(dest_codec, Exception):
            raise dest_codec
        url = self._build_url(target)
        body = self._encoder.encode(req)
        content_type = self._encoder.content_type()
        accept_header = dest_codec.content_type()

        def finalizer(ctx: Context) -> Context:
            out_ctx = Context(url, self._endpoint.protocol, "client")
            headers = {
                **self._headers(content_type, accept_header),
                **ctx.params,
            }
            http_res: BaseHTTPResponse
            try:
                http_res = self._pool.request(
                    method="POST",
                    url=url,
                    headers=headers,
                    body=body,
                    preload_content=False,
                )
            except MaxRetryError as e:
                raise Unreachable(url, e.url or "Unreachable") from e
            try:
                out_ctx.params = http_res.headers
                if http_res.status < 200 or http_res.status >= 300:
                    raise self._decode_error(http_res, http_res.read())
                with open(dest, "wb") as out:
                    for chunk in http_res.stream():
                        out.write(chunk)
                return out_ctx
            finally:
                http_res.release_conn()

        in_ctx = Context(url, self._endpoint.protocol, "client")
        self.exec(in_ctx, finalizer)

    def _build_url(self, target: str) -> str:
        return self._endpoint.child(target).stringify()

    def _codec_for_path(self, path: FilePath) -> FileCodec | Exception:
        ext = pathlib.Path(os.fspath(path)).suffix.lstrip(".").lower()
        codec = self._codecs_by_extension.get(ext)
        if codec is None:
            return ValueError(
                f"no codec registered for file extension {ext!r} (path: {os.fspath(path)!r})"
            )
        return codec

    def _headers(self, content_type: str, accept: str) -> dict[str, str]:
        return {
            _CONTENT_TYPE_HEADER_KEY: content_type,
            "Accept": accept,
        }

    def _typed_response_request(
        self,
        target: str,
        body: bytes | IO[bytes] | Iterable[bytes] | None,
        content_type: str,
        res_t: type[RS],
    ) -> RS:
        url = self._build_url(target)
        in_ctx = Context(url, self._endpoint.protocol, "client")
        res_container: list[RS | None] = [None]

        def finalizer(ctx: Context) -> Context:
            out_ctx = Context(url, self._endpoint.protocol, "client")
            headers = {
                **self._headers(content_type, self._accept_header),
                **ctx.params,
            }
            try:
                http_res = self._pool.request(
                    method="POST", url=url, headers=headers, body=body
                )
            except MaxRetryError as e:
                raise Unreachable(url, e.url or "Unreachable") from e
            out_ctx.params = http_res.headers
            if not 200 <= http_res.status < 300:
                raise self._decode_error(http_res, http_res.data)
            if http_res.data is None or len(http_res.data) == 0:
                raise ValueError(f"expected a non-empty response body from {url!r}")
            decoder = self._resolve_decoder(http_res)
            if isinstance(decoder, Exception):
                raise decoder
            res_container[0] = decoder.decode(http_res.data, res_t)
            return out_ctx

        self.exec(in_ctx, finalizer)
        res = res_container[0]
        assert res is not None
        return res

    def _resolve_decoder(self, http_res: BaseHTTPResponse) -> FileCodec | Exception:
        ct = http_res.headers.get(_CONTENT_TYPE_HEADER_KEY, "")
        decoder = self._decoders_by_content_type.get(ct.split(";", 1)[0].strip())
        if decoder is None:
            return ValueError(f"no decoder registered for response Content-Type {ct!r}")
        return decoder

    def _decode_error(self, http_res: BaseHTTPResponse, data: bytes) -> Exception:
        decoder = self._resolve_decoder(http_res)
        if isinstance(decoder, Exception):
            return decoder
        try:
            payload = decoder.decode(data, ExceptionPayload)
        except Exception:
            return ValueError(f"undecodable error response: {data!r}")
        decoded = decode_exception(payload)
        if decoded is None:
            return ValueError(f"undecodable error response: {data!r}")
        return decoded
