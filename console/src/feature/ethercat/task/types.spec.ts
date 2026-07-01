// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";

describe("EtherCAT Task statusData", () => {
  describe("readStatusDataZ", () => {
    it("should accept null", () => {
      expect(EtherCAT.Task.READ_SCHEMAS.statusData.safeParse(null).success).toBe(true);
    });
    it("should accept undefined", () => {
      expect(EtherCAT.Task.READ_SCHEMAS.statusData.safeParse(undefined).success).toBe(
        true,
      );
    });
    it("should accept a valid status object", () => {
      const result = EtherCAT.Task.READ_SCHEMAS.statusData.safeParse({
        running: true,
        message: "ok",
        errors: [{ message: "err", path: "/dev" }],
      });
      expect(result.success).toBe(true);
    });
  });

  describe("writeStatusDataZ", () => {
    it("should accept null", () => {
      expect(EtherCAT.Task.WRITE_SCHEMAS.statusData.safeParse(null).success).toBe(true);
    });
    it("should accept undefined", () => {
      expect(EtherCAT.Task.WRITE_SCHEMAS.statusData.safeParse(undefined).success).toBe(
        true,
      );
    });
    it("should accept a valid status object", () => {
      const result = EtherCAT.Task.WRITE_SCHEMAS.statusData.safeParse({
        running: false,
        message: "ok",
        errors: [{ message: "err", path: "/dev" }],
      });
      expect(result.success).toBe(true);
    });
  });
});
