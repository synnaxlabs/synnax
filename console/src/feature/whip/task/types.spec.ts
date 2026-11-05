// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { WHIP } from "@/feature/whip";

const KEY = "0b6e33b3-6c63-4b3f-9f35-7f3e2d1c0a9b";

describe("WHIP task types", () => {
  describe("publishURL", () => {
    it("should build an http URL for a Core without TLS", () => {
      expect(
        WHIP.Task.publishURL({ host: "localhost", port: 9090, secure: false }, KEY),
      ).toBe(`http://localhost:9090/api/v1/whip/${KEY}`);
    });

    it("should build an https URL for a Core with TLS", () => {
      expect(
        WHIP.Task.publishURL({ host: "core.lab", port: 443, secure: true }, KEY),
      ).toBe(`https://core.lab:443/api/v1/whip/${KEY}`);
    });
  });
});
