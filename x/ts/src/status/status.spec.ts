// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { id } from "@/id";
import { status } from "@/status";
import { TimeStamp } from "@/telem";

describe("status", () => {
  describe("create", () => {
    it("should create a status", () => {
      const s = status.create({ variant: "success", message: "test" });
      expect(s.key).toHaveLength(id.LENGTH);
      expect(s.time).toBeInstanceOf(TimeStamp);
      expect(s.time.beforeEq(TimeStamp.now())).toBe(true);
    });
  });

  describe("keepVariants", () => {
    it("should return undefined when variant is null", () => {
      expect(status.keepVariants(undefined, "success")).toBeUndefined();
    });

    it("should return undefined when variant is not in keep list", () => {
      expect(status.keepVariants("error", "success")).toBeUndefined();
      expect(status.keepVariants("error", ["success", "info"])).toBeUndefined();
    });

    it("should return variant when it matches single keep variant", () => {
      expect(status.keepVariants("success", "success")).toBe("success");
    });

    it("should return variant when it is in keep array", () => {
      expect(status.keepVariants("success", ["success", "info"])).toBe("success");
      expect(status.keepVariants("info", ["success", "info"])).toBe("info");
    });

    it("should return undefined when keep is empty array", () => {
      expect(status.keepVariants("success", [])).toBeUndefined();
    });
  });

  describe("removeVariants", () => {
    it("should return undefined when variant is null", () => {
      expect(status.removeVariants(undefined, "success")).toBeUndefined();
    });

    it("should return undefined when variant matches single remove variant", () => {
      expect(status.removeVariants("success", "success")).toBeUndefined();
    });

    it("should return undefined when variant is in remove array", () => {
      expect(status.removeVariants("success", ["success", "error"])).toBeUndefined();
      expect(status.removeVariants("error", ["success", "error"])).toBeUndefined();
    });

    it("should return variant when it does not match single remove variant", () => {
      expect(status.removeVariants("success", "error")).toBe("success");
    });

    it("should return variant when it is not in remove array", () => {
      expect(status.removeVariants("warning", ["success", "error"])).toBe("warning");
      expect(status.removeVariants("info", ["success", "error"])).toBe("info");
    });

    it("should return variant when remove is empty array", () => {
      expect(status.removeVariants("success", [])).toBe("success");
    });
  });
});
