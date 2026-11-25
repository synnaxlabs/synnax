// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CSS } from "@/css";

const countMutations = (element: HTMLElement, fn: () => void): number => {
  const observer = new MutationObserver(() => {});
  observer.observe(element, { attributes: true });
  fn();
  const count = observer.takeRecords().length;
  observer.disconnect();
  return count;
};

describe("vars", () => {
  describe("applyVars", () => {
    it("should set every variable in one style mutation", () => {
      const el = document.createElement("div");
      const mutations = countMutations(el, () =>
        CSS.applyVars(el, { "--pluto-a": "red", "--pluto-b": 2, "--pluto-c": "1rem" }),
      );
      expect(mutations).toBe(1);
      expect(el.style.getPropertyValue("--pluto-a")).toBe("red");
      expect(el.style.getPropertyValue("--pluto-b")).toBe("2");
      expect(el.style.getPropertyValue("--pluto-c")).toBe("1rem");
    });

    it("should keep the other inline properties", () => {
      const el = document.createElement("div");
      el.style.setProperty("color", "blue", "important");
      el.style.setProperty("--pluto-a", "red");
      CSS.applyVars(el, { "--pluto-a": "green" });
      expect(el.style.getPropertyValue("color")).toBe("blue");
      expect(el.style.getPropertyPriority("color")).toBe("important");
      expect(el.style.getPropertyValue("--pluto-a")).toBe("green");
    });

    it("should skip undefined values", () => {
      const el = document.createElement("div");
      el.style.setProperty("--pluto-a", "red");
      CSS.applyVars(el, { "--pluto-a": undefined });
      expect(el.style.getPropertyValue("--pluto-a")).toBe("red");
    });
  });

  describe("removeVars", () => {
    it("should remove every prefixed property in one style mutation", () => {
      const el = document.createElement("div");
      el.style.setProperty("color", "blue");
      el.style.setProperty("--pluto-a", "red");
      el.style.setProperty("--pluto-b", "green");
      el.style.setProperty("--other-a", "black");
      const mutations = countMutations(el, () => CSS.removeVars(el, "--pluto"));
      expect(mutations).toBe(1);
      expect(Array.from(el.style)).toEqual(["color", "--other-a"]);
    });
  });
});
