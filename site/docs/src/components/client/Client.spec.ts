// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getFromURL, getStored, setInURL } from "@/components/client/Client";

describe("Client", () => {
  beforeEach(() => {
    localStorage.clear();
    window.history.replaceState({}, "", "/reference/channels/create");
  });

  afterEach(() => vi.restoreAllMocks());

  describe("getStored", () => {
    it("should return null when no choice is stored", () => {
      expect(getStored()).toBeNull();
    });

    it("should return null for a stored value that is not a client", () => {
      localStorage.setItem("docs.client", "rust");
      expect(getStored()).toBeNull();
    });

    it("should return null when storage is blocked", () => {
      vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
        throw new Error("SecurityError");
      });
      expect(getStored()).toBeNull();
    });
  });

  describe("setInURL", () => {
    it("should put the client in the URL and remember it", () => {
      setInURL("python");
      expect(getFromURL()).toBe("python");
      expect(getStored()).toBe("python");
    });

    it("should replace the history entry when asked", () => {
      const length = window.history.length;
      setInURL("typescript", true);
      expect(window.history.length).toBe(length);
      expect(getFromURL()).toBe("typescript");
    });

    it("should still update the URL when storage is blocked", () => {
      vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new Error("QuotaExceededError");
      });
      setInURL("cpp");
      expect(getFromURL()).toBe("cpp");
    });
  });
});
