// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Transport } from "@/transport";

/**
 * The set of body types accepted by FileTransport.upload. A ReadableStream or Blob is
 * streamed to the server without being buffered into memory; ArrayBufferView and string
 * bodies are sent as-is.
 */
export type UploadBody =
  ReadableStream<Uint8Array> | Blob | ArrayBufferView<ArrayBuffer> | string;

/**
 * The wire encodings a FileTransport can transfer.
 */
export type FileEncoding = "JSON";

/**
 * Options shared by FileTransport.upload and FileTransport.download. Carries the wire
 * encoding of the transferred bytes and is the extension point for any future
 * per-transfer settings.
 */
export interface FileOptions {
  /**
   * The wire encoding of the transferred bytes.
   *
   * On upload it describes the request body and is sent as Content-Type; the body has a
   * single representation, so pass a single encoding.
   */
  encoding: FileEncoding;
  /**
   * Request params appended to the URL as freighterctx-prefixed query parameters. The
   * body is the raw file bytes, so per-transfer metadata (e.g. the source file's name)
   * travels out-of-band here. Keys are the bare param names — the transport adds the
   * prefix — and undefined values are omitted.
   */
  params?: Record<string, string | undefined>;
}

/**
 * FileTransport streams request and response bodies to and from a server when a payload
 * could be too large to buffer in memory. Unlike UnaryClient, which encodes and decodes
 * a typed payload on each side, FileTransport leaves one side of the exchange as a raw
 * byte stream: upload streams the request body up, and download streams the response
 * body back. It is environment-agnostic — the byte stream is a standard Web
 * ReadableStream, which exists in browsers, Node 18+, and Tauri webviews alike.
 */
export interface FileTransport extends Transport {
  /**
   * Streams body to target as the request body and decodes the typed response.
   *
   * @param target - The target route to send the request to.
   * @param body - The request body, streamed to the server without buffering when it is
   * a ReadableStream or Blob.
   * @param options - The transfer options, including the encoding of body.
   * @param resSchema - The schema to validate the decoded response against.
   * @returns the decoded response.
   * @throws Unreachable: if the target cannot be reached.
   * @throws Error: if the server returns an error.
   */
  upload: <RS extends z.ZodType>(
    target: string,
    body: UploadBody,
    options: FileOptions,
    resSchema: RS,
  ) => Promise<z.infer<RS>>;

  /**
   * Sends the typed request req to target and returns the response body as a byte
   * stream the caller can pipe wherever it likes (a file, a download, an in-memory
   * sink) without buffering the whole payload.
   *
   * @param target - The target route to send the request to.
   * @param req - The typed request payload.
   * @param reqSchema - The schema to validate and encode the request with.
   * @param options - The transfer options, including the desired response encoding.
   * @returns the response body as a ReadableStream of bytes.
   * @throws Unreachable: if the target cannot be reached.
   * @throws Error: if the server returns an error.
   */
  download: <RQ extends z.ZodType>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    options: FileOptions,
  ) => Promise<ReadableStream<Uint8Array>>;
}
