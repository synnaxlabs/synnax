// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UploadBody } from "@synnaxlabs/freighter";

import { supportsRequestStreams } from "@/platform/runtime/requestStreams";

/**
 * Adapts body to the most performant form the runtime's fetch can stream. Case 1:
 * Files, Blobs, and bytes pass through — every engine streams a file-backed Blob from
 * disk. Case 2: a ReadableStream passes through where fetch supports streaming request
 * bodies. Case 3: elsewhere the stream is buffered into a single Blob.
 */
export const uploadBody = async (body: UploadBody): Promise<UploadBody> => {
  if (!(body instanceof ReadableStream) || supportsRequestStreams()) return body;
  return await new Response(body).blob();
};
