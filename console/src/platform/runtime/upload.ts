// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UploadBody } from "@synnaxlabs/freighter";

// Feature-detects streaming request bodies: constructing a Request with a stream body
// reads duplex and leaves Content-Type unset only on engines that support them
// (Chromium family, Node); elsewhere the stream is stringified.
const supportsRequestStreams = (() => {
  let duplexAccessed = false;
  const hasContentType = new Request("http://localhost/", {
    body: new ReadableStream(),
    method: "POST",
    get duplex() {
      duplexAccessed = true;
      return "half";
    },
  } as RequestInit).headers.has("Content-Type");
  return duplexAccessed && !hasContentType;
})();

/**
 * Adapts body to the most performant form the runtime's fetch can stream. Case 1:
 * Files, Blobs, and bytes pass through — every engine streams a file-backed Blob from
 * disk. Case 2: a ReadableStream passes through where fetch supports streaming request
 * bodies. Case 3: elsewhere the stream is buffered into a single Blob.
 */
export const uploadBody = async (body: UploadBody): Promise<UploadBody> => {
  if (!(body instanceof ReadableStream) || supportsRequestStreams) return body;
  return await new Response(body).blob();
};
