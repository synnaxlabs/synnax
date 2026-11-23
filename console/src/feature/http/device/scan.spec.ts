// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { HTTP } from "@/feature/http";

describe("HTTP Scan Task", () => {
  describe("config", () => {
    it("should accept null", () => {
      expect(z.validate(HTTP.Device.SCAN_SCHEMAS.config, null)).toBe(true);
    });
    it("should accept undefined", () => {
      expect(z.validate(HTTP.Device.SCAN_SCHEMAS.config, undefined)).toBe(true);
    });
  });
  describe("statusData", () => {
    it("should accept null", () => {
      expect(z.validate(HTTP.Device.SCAN_SCHEMAS.statusData, null)).toBe(true);
    });
    it("should accept undefined", () => {
      expect(z.validate(HTTP.Device.SCAN_SCHEMAS.statusData, undefined)).toBe(true);
    });
  });
});
