// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Account } from "@/session/account";

const LINKED: Account.LinkPayload = {
  activation: "act",
  secret: "secret",
  email: "someone@example.com",
};

describe("account slice", () => {
  it("should start unlinked", () => {
    expect(Account.ZERO_SLICE_STATE).toEqual({ version: 0 });
  });

  describe("beginSignIn", () => {
    it("should hold the minted state", () => {
      const next = Account.reducer(Account.ZERO_SLICE_STATE, Account.beginSignIn("s"));
      expect(next.pending).toBe("s");
    });
  });

  describe("link", () => {
    it("should store the link and drop the pending state", () => {
      const begun = Account.reducer(Account.ZERO_SLICE_STATE, Account.beginSignIn("s"));
      const next = Account.reducer(begun, Account.link(LINKED));
      expect(next).toEqual({ version: 0, ...LINKED });
    });
  });

  describe("clear", () => {
    it("should forget the link", () => {
      const linked = Account.reducer(Account.ZERO_SLICE_STATE, Account.link(LINKED));
      expect(Account.reducer(linked, Account.clear())).toEqual(
        Account.ZERO_SLICE_STATE,
      );
    });
  });

  it("should read a stored link", () => {
    expect(Account.sliceStateZ.parse({ version: 0, ...LINKED })).toEqual({
      version: 0,
      ...LINKED,
    });
  });
});
