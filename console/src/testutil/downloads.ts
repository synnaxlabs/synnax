// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vi } from "vitest";

export const MOCK_OBJECT_URL = "blob:mock";

export interface CapturedDownloads {
  /** Anchors whose click was captured, in download order. */
  anchors: HTMLAnchorElement[];
  /** Blobs handed to URL.createObjectURL, in download order. */
  blobs: Blob[];
  /** Object URLs passed to URL.revokeObjectURL, in revocation order. */
  revoked: string[];
}

/**
 * captureBrowserDownloads stubs the object-URL and anchor-click primitives jsdom lacks
 * so browser download paths complete, recording each downloaded anchor, blob, and
 * revoked URL for assertion. Installs spies; callers restore them via
 * vi.restoreAllMocks() in afterEach.
 */
export const captureBrowserDownloads = (): CapturedDownloads => {
  const anchors: HTMLAnchorElement[] = [];
  const blobs: Blob[] = [];
  const revoked: string[] = [];
  vi.spyOn(URL, "createObjectURL").mockImplementation((data) => {
    if (data instanceof Blob) blobs.push(data);
    return MOCK_OBJECT_URL;
  });
  vi.spyOn(URL, "revokeObjectURL").mockImplementation((url) => {
    revoked.push(url);
  });
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    anchors.push(this);
  });
  return { anchors, blobs, revoked };
};
