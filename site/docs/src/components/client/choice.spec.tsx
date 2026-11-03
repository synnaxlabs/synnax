// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Client } from "@/components/client";

describe("client choice", () => {
  it("should show the TypeScript form of inline values for TypeScript", () => {
    document.body.innerHTML = renderToString(
      <Client.Var py="set_authority" ts="setAuthority" />,
    );
    Client.QUERY.onChange?.("typescript");
    expect(document.body.textContent).toBe("setAuthority");
  });

  it("should show the Python form of inline values for any other client", () => {
    document.body.innerHTML = renderToString(
      <Client.Var py="set_authority" ts="setAuthority" />,
    );
    Client.QUERY.onChange?.("typescript");
    Client.QUERY.onChange?.("console");
    expect(document.body.textContent).toBe("set_authority");
  });

  describe("storage", () => {
    beforeEach(() => localStorage.clear());

    afterEach(() => vi.restoreAllMocks());

    it("should start with no choice when none is stored", () => {
      expect(Client.QUERY.initial?.()).toBeNull();
    });

    it("should remember a choice for later pages", () => {
      Client.QUERY.onChange?.("python");
      expect(Client.QUERY.initial?.()).toBe("python");
    });

    it("should ignore a stored value that is not a client", () => {
      localStorage.setItem("docs.client", "rust");
      expect(Client.QUERY.initial?.()).toBeNull();
    });

    it("should start with no choice when storage is blocked", () => {
      vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
        throw new Error("SecurityError");
      });
      expect(Client.QUERY.initial?.()).toBeNull();
    });

    it("should still apply a choice when storage is blocked", () => {
      document.body.innerHTML = renderToString(
        <Client.Var py="set_authority" ts="setAuthority" />,
      );
      vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new Error("QuotaExceededError");
      });
      Client.QUERY.onChange?.("typescript");
      expect(document.body.textContent).toBe("setAuthority");
    });
  });
});
