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
from collections.abc import Iterable
from typing import IO, Any

import urllib3
from urllib3 import PoolManager, Retry
from urllib3.exceptions import MaxRetryError
from urllib3.response import BaseHTTPResponse

from freighter.codec import Codec
from freighter.context import Context
from freighter.exceptions import Unreachable
from freighter.transport import RQ, RS, MiddlewareCollector
from freighter.url import URL
from x.exceptions import ExceptionPayload, decode_exception
from x.file import FilePath, stream_to_file


_FILE_CONTENT_TYPES: dict[str, str] = {
    "json": "application/json",
    "msgpack": "application/msgpack",
    "toml": "application/toml",
    "yaml": "application/yaml",
}


def _file_content_type(path: FilePath) -> str:
    """:returns: the wire content type negotiated for path's extension.

    :raises ValueError: if the extension has no registered content type.
    """
    ext = pathlib.Path(os.fspath(path)).suffix.lstrip(".").lower()
    content_type = _FILE_CONTENT_TYPES.get(ext)
    if content_type is None:
        raise ValueError(
            f"no content type registered for file extension {ext!r} "
            f"(path: {os.fspath(path)!r})"
        )
    return content_type


class HTTPClient(MiddlewareCollector):
    """
    HTTPClient is a urllib3-backed transport implementing UnaryClient and FileTransport.

    A single codec encodes and decodes the typed request and response payloads. The
    Content-Type and Accept headers for a streamed file body are derived from the file
    path's extension at call time — the body is streamed verbatim, so the client never
    (de)serializes the file and supports any on-disk format the server understands.

    - send: typed request, typed response, both via the codec (UnaryClient).
    - upload: a file path as the request body — the Content-Type is derived from the
      path extension — with a typed response decoded via the codec. Bytes are streamed
      from disk in fixed-size blocks so the full body is never held in memory
      (FileTransport).
    - download: a typed request encoded via the codec, with the response streamed
      directly into a destination file path whose extension drives the Accept header
      (FileTransport).
    """

    _pool: PoolManager
    _endpoint: URL
    _codec: Codec

    def __init__(
        self,
        url: URL,
        codec: Codec,
        secure: bool = False,
        **kwargs: Any,
    ) -> None:
        """
        :param url: The base URL for the client.
        :param codec: The codec used to encode and decode typed request and response
            payloads.
        :param secure: Whether to use HTTPS.
        """
        super().__init__()
        self._endpoint = url
        self._endpoint.protocol = "https" if secure else "http"
        self._codec = codec
        self._pool = PoolManager(cert_reqs="CERT_NONE", **kwargs)
        urllib3.disable_warnings()

    def send(self, target: str, req: RQ, res_t: type[RS]) -> RS:
        """Implements the UnaryClient protocol — typed request, typed response."""
        return self._typed_response_request(
            target=target,
            body=self._codec.encode(req),
            content_type=self._codec.content_type(),
            res_t=res_t,
        )

    def upload(self, target: str, req: FilePath, res_t: type[RS]) -> RS:
        """
        Streams the file at req to target and decodes the response into res_t. The file
        object is handed to urllib3 as the request body and read in fixed-size blocks,
        so the full body never has to fit in memory. The Content-Type is derived from
        the file path's extension, while the typed response is decoded via the codec —
        the two need not agree.

        Retries are disabled for uploads because an upload mutates server state and is
        not assumed to be idempotent: a transient failure that occurs after the server
        has begun applying the import could replay it on retry and double-apply the
        data. (urllib3 would in fact rewind the seekable file body and resend the full
        request via set_file_position/rewind_body, so the stream itself is replayable;
        the concern is request semantics, not the body.) A failed upload surfaces to the
        caller instead, leaving the retry decision to the caller.
        """
        with open(req, "rb") as f:
            return self._typed_response_request(
                target=target,
                body=f,
                content_type=_file_content_type(req),
                res_t=res_t,
                retries=False,
            )

    def download(self, target: str, req: RQ, dest: FilePath) -> None:
        """
        Sends req to target and streams the response body directly into dest, without
        buffering the full body in memory; the Accept header is derived from the dest
        path's extension (e.g. a .json destination requests application/json), while the
        typed request is encoded via the codec.

        The body is streamed into a temporary file alongside dest and atomically renamed
        into place on success, so dest only ever holds the previous contents or the
        complete new contents — a mid-stream failure leaves any existing dest untouched.
        """
        url = self._build_url(target)

        def finalizer(ctx: Context) -> Context:
            http_res, out_ctx = self._request(
                ctx,
                url=url,
                content_type=self._codec.content_type(),
                accept=_file_content_type(dest),
                body=self._codec.encode(req),
                preload_content=False,
            )
            try:
                stream_to_file(http_res.stream(), dest)
                return out_ctx
            finally:
                http_res.release_conn()

        in_ctx = Context(url, self._endpoint.protocol, "client")
        self.exec(in_ctx, finalizer)

    def _build_url(self, target: str) -> str:
        return self._endpoint.child(target).stringify()

    def _typed_response_request(
        self,
        target: str,
        body: bytes | IO[bytes] | Iterable[bytes] | None,
        content_type: str,
        res_t: type[RS],
        retries: Retry | bool | None = None,
    ) -> RS:
        url = self._build_url(target)
        in_ctx = Context(url, self._endpoint.protocol, "client")
        res_container: list[RS | None] = [None]

        def finalizer(ctx: Context) -> Context:
            http_res, out_ctx = self._request(
                ctx,
                url=url,
                content_type=content_type,
                accept=self._codec.content_type(),
                body=body,
                preload_content=True,
                retries=retries,
            )
            if http_res.data is None or len(http_res.data) == 0:
                raise ValueError(f"expected a non-empty response body from {url!r}")
            res_container[0] = self._codec.decode(http_res.data, res_t)
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
        retries: Retry | bool | None = None,
    ) -> tuple[BaseHTTPResponse, Context]:
        """Issues a POST to url and returns the response alongside an outgoing context.

        Raises Unreachable on a transport failure and the decoded server error on a
        non-2xx status. On success the response body has not been consumed; when
        preload_content is False the caller owns release_conn. A retries value of None
        defers to the pool default; pass False to disable retries (e.g. for an
        unreplayable streaming body).
        """
        out_ctx = Context(url, self._endpoint.protocol, "client")
        headers = {"Content-Type": content_type, "Accept": accept, **ctx.params}
        try:
            http_res = self._pool.request(
                method="POST",
                url=url,
                headers=headers,
                body=body,
                preload_content=preload_content,
                retries=retries,
            )
        except MaxRetryError as e:
            raise Unreachable(url, e.url or "Unreachable") from e
        out_ctx.params = http_res.headers
        if not 200 <= http_res.status < 300:
            try:
                try:
                    payload = self._codec.decode(http_res.data, ExceptionPayload)
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
