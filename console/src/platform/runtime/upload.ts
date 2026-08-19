// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UploadBody } from "@synnaxlabs/freighter";

/**
 * Adapts body to a form the runtime's fetch can send on any connection. Files, Blobs,
 * and bytes pass through — every engine streams a file-backed Blob from disk. A
 * ReadableStream is buffered into a single Blob: Chromium sends streaming request
 * bodies only over HTTP/2 or HTTP/3, and the negotiated protocol is unknowable before
 * the request, so streaming would fail against any HTTP/1.1 Core.
 */
export const uploadBody = async (body: UploadBody): Promise<UploadBody> => {
  if (!(body instanceof ReadableStream)) return body;
  return await new Response(body).blob();
};
