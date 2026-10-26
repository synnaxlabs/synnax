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

const LINKED: Account.Link = {
  secret: "secret",
  email: "someone@example.com",
  user: "user_a",
};

describe("account slice", () => {
  it("should start unlinked with analytics refused", () => {
    expect(Account.ZERO_SLICE_STATE).toEqual({ version: 0, analyticsAllowed: false });
  });

  describe("beginLogin", () => {
    it("should hold the minted state", () => {
      const next = Account.reducer(Account.ZERO_SLICE_STATE, Account.beginLogin("s"));
      expect(next.pending).toBe("s");
    });
  });

  describe("link", () => {
    it("should store the link and drop the pending state", () => {
      const begun = Account.reducer(Account.ZERO_SLICE_STATE, Account.beginLogin("s"));
      const next = Account.reducer(
        begun,
        Account.link({ ...LINKED, analyticsAllowed: false }),
      );
      expect(next).toEqual({ version: 0, link: LINKED, analyticsAllowed: false });
    });

    it("should take the analytics choice from the link", () => {
      const next = Account.reducer(
        Account.ZERO_SLICE_STATE,
        Account.link({ ...LINKED, analyticsAllowed: true }),
      );
      expect(next.analyticsAllowed).toBe(true);
    });

    it("should replace an earlier analytics choice", () => {
      const allowed = Account.reducer(
        Account.ZERO_SLICE_STATE,
        Account.setAnalyticsAllowed(true),
      );
      const next = Account.reducer(
        allowed,
        Account.link({ ...LINKED, analyticsAllowed: false }),
      );
      expect(next.analyticsAllowed).toBe(false);
    });
  });

  describe("setAnalyticsAllowed", () => {
    it("should keep the link when the choice changes", () => {
      const linked = Account.reducer(
        Account.ZERO_SLICE_STATE,
        Account.link({ ...LINKED, analyticsAllowed: true }),
      );
      const next = Account.reducer(linked, Account.setAnalyticsAllowed(false));
      expect(next).toEqual({ version: 0, link: LINKED, analyticsAllowed: false });
    });
  });

  describe("clear", () => {
    it("should forget the link and refuse analytics", () => {
      const linked = Account.reducer(
        Account.ZERO_SLICE_STATE,
        Account.link({ ...LINKED, analyticsAllowed: true }),
      );
      expect(Account.reducer(linked, Account.clear())).toEqual(
        Account.ZERO_SLICE_STATE,
      );
    });
  });

  it("should read a link stored before the analytics choice existed", () => {
    expect(Account.sliceStateZ.parse({ version: 0, link: LINKED })).toEqual({
      version: 0,
      link: LINKED,
      analyticsAllowed: false,
    });
  });
});
