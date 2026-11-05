// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { RTSP } from "@/feature/rtsp";

const config = (url: string) => ({
  ...RTSP.Task.READ_SCHEMAS.config.parse({}),
  url,
});

describe("RTSP task types", () => {
  describe("deployReadConfigZ", () => {
    it("should accept an rtsp or rtsps URL", () => {
      for (const url of ["rtsp://camera.local/stream", "rtsps://camera.local:322/s"])
        expect(RTSP.Task.deployReadConfigZ.safeParse(config(url)).success).toBe(true);
    });

    it("should reject an empty URL or one with another scheme", () => {
      for (const url of ["", "http://camera.local/stream", "rtsp://"]) {
        const result = RTSP.Task.deployReadConfigZ.safeParse(config(url));
        expect(result.success).toBe(false);
        expect(result.error?.issues[0].message).toBe(
          "URL must start with rtsp:// or rtsps://",
        );
      }
    });
  });
});
