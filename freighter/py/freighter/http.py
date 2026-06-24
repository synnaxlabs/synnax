#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import pathlib
import tempfile
from collections.abc import Iterable, Sequence
from typing import IO, Any, NoReturn

import urllib3
from urllib3 import PoolManager
from urllib3.exceptions import MaxRetryError
from urllib3.response import BaseHTTPResponse

from freighter.context import Context
from freighter.exceptions import Unreachable
from freighter.file import FileCodec, FilePath
from freighter.transport import RQ, RS, MiddlewareCollector
from freighter.url import URL
from x.exceptions import ExceptionPayload, decode_exception

_CONTENT_TYPE_HEADER_KEY = "Content-Type"


def stream_to_file(chunks: Iterable[bytes], dest: FilePath) -> None:
    """Writes a stream of byte chunks into dest atomically.

    The chunks are streamed into a temporary file alongside dest and the temp file is
    renamed into place only once the full stream is consumed, so dest is observed to
    hold either its previous contents or the complete new contents — never a partial
    write. A failure partway through removes the temp file and leaves any existing dest
    untouched.

    :param chunks: an iterable of byte chunks to write in order.
    :param dest: the destination file path.
    """
    dest_path = pathlib.Path(os.fspath(dest))
    fd, tmp_name = tempfile.mkstemp(dir=dest_path.parent, suffix=".part")
    try:
        with os.fdopen(fd, "wb") as out:
            for chunk in chunks:
                out.write(chunk)
        os.replace(tmp_name, dest_path)
    except BaseException:
        os.unlink(tmp_name)
        raise


class HTTPClient(MiddlewareCollector):
    """
    HTTPClient is a urllib3-backed transport implementing UnaryClient and FileClient:

    - send: typed request, typed response, both via the configured encoder/decoders
      (UnaryClient).
    - upload: a file path as the request body, with a typed response. Bytes are streamed
      from disk in fixed-size blocks so the full body is never held in memory; the
      Content-Type is inferred from the path's extension (FileClient).
    - download: a typed request, with the response streamed directly into a destination
      file path; the Accept header is derived from the destination extension
      (FileClient).
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
        Streams the file at req to target and decodes the response into res_t. The file
        object is handed to urllib3 as the request body and read in fixed-size blocks,
        so the full body never has to fit in memory. The Content-Type is inferred from
        the file extension via the client's registered codecs.
        """
        codec = self._codec_for_path(req)
        with open(req, "rb") as f:
            return self._typed_response_request(
                target=target,
                body=f,
                content_type=codec.content_type(),
                res_t=res_t,
            )

    def download(self, target: str, req: RQ, dest: FilePath) -> None:
        """
        Sends req to target and streams the response body directly into dest, without
        buffering the full body in memory; the Accept header is derived from the dest
        extension via the client's registered codecs (e.g., a .json destination requests
        application/json), so the on-disk format and the negotiated wire format are
        guaranteed to match.

        The body is streamed into a temporary file alongside dest and atomically renamed
        into place on success, so dest only ever holds the previous contents or the
        complete new contents — a mid-stream failure leaves any existing dest untouched.
        """
        dest_codec = self._codec_for_path(dest)
        url = self._build_url(target)

        def finalizer(ctx: Context) -> Context:
            http_res, out_ctx = self._request(
                ctx,
                url=url,
                content_type=self._encoder.content_type(),
                accept=dest_codec.content_type(),
                body=self._encoder.encode(req),
                preload_content=False,
            )
            try:
                stream_to_file(http_res.stream(), dest)
                return out_ctx
            finally:
                http_res.release_conn()

        in_ctx = Context(url, self._endpoint.protocol, "client")
        self.exec(in_ctx, finalizer)
        return None

    def _build_url(self, target: str) -> str:
        return self._endpoint.child(target).stringify()

    def _codec_for_path(self, path: FilePath) -> FileCodec:
        ext = pathlib.Path(os.fspath(path)).suffix.lstrip(".").lower()
        codec = self._codecs_by_extension.get(ext)
        if codec is None:
            raise ValueError(
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
            http_res, out_ctx = self._request(
                ctx,
                url=url,
                content_type=content_type,
                accept=self._accept_header,
                body=body,
                preload_content=True,
            )
            if http_res.data is None or len(http_res.data) == 0:
                raise ValueError(f"expected a non-empty response body from {url!r}")
            decoder = self._resolve_decoder(http_res)
            res_container[0] = decoder.decode(http_res.data, res_t)
            return out_ctx

        self.exec(in_ctx, finalizer)
        res = res_container[0]
        assert res is not None
        return res

    def _request(
        self,
        ctx: Context,
        url: str,
        content_type: str,
        accept: str,
        body: bytes | IO[bytes] | Iterable[bytes] | None,
        preload_content: bool,
    ) -> tuple[BaseHTTPResponse, Context]:
        """Issues a POST to url and returns the response alongside an outgoing context.

        Raises Unreachable on a transport failure and the decoded server error on a
        non-2xx status. On success the response body has not been consumed; when
        preload_content is False the caller owns release_conn.
        """
        out_ctx = Context(url, self._endpoint.protocol, "client")
        headers = {**self._headers(content_type, accept), **ctx.params}
        try:
            http_res = self._pool.request(
                method="POST",
                url=url,
                headers=headers,
                body=body,
                preload_content=preload_content,
            )
        except MaxRetryError as e:
            raise Unreachable(url, e.url or "Unreachable") from e
        out_ctx.params = http_res.headers
        if not 200 <= http_res.status < 300:
            try:
                decoder = self._resolve_decoder(http_res)
                try:
                    payload = decoder.decode(http_res.data, ExceptionPayload)
                except Exception as e:
                    raise ValueError(
                        f"undecodable error response: {http_res.data!r}"
                    ) from e
                decoded = decode_exception(payload)
                if decoded is None:
                    raise ValueError(f"undecodable error response: {http_res.data!r}")
                raise decoded
            finally:
                if not preload_content:
                    http_res.release_conn()
        return http_res, out_ctx

    def _resolve_decoder(self, http_res: BaseHTTPResponse) -> FileCodec:
        ct = http_res.headers.get(_CONTENT_TYPE_HEADER_KEY, "")
        decoder = self._decoders_by_content_type.get(ct.split(";", 1)[0].strip())
        if decoder is None:
            raise ValueError(f"no decoder registered for response Content-Type {ct!r}")
        return decoder
