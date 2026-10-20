// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

const PNG_SIGNATURE = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];
const IHDR = [0, 0, 0, 13, 0x49, 0x48, 0x44, 0x52];

const header = (width: number, height: number): Uint8Array<ArrayBuffer> => {
  const bytes = new Uint8Array(33);
  bytes.set([...PNG_SIGNATURE, ...IHDR]);
  const view = new DataView(bytes.buffer);
  view.setUint32(16, width);
  view.setUint32(20, height);
  return bytes;
};

/**
 * Answers every fetch with the header of a PNG of the given size, so that Astro reads
 * remote image sizes without the network. Returns the fetch mock.
 */
export const stubImageSize = (width: number, height: number): Mock => {
  const fetch = vi.fn(async (request: Request) => {
    const response = new Response(header(width, height));
    Object.defineProperty(response, "url", { value: request.url });
    return response;
  });
  vi.stubGlobal("fetch", fetch);
  return fetch;
};
