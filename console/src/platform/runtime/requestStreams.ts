// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Feature-detects streaming request bodies: constructing a Request with a stream body
 * reads duplex and leaves Content-Type unset only on engines that support them
 * (Chromium family, Node); elsewhere the stream is stringified.
 */
export const supportsRequestStreams = (): boolean => {
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
};
