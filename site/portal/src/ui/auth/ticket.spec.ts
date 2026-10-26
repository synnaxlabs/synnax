// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { readTicket } from "@/ui/auth/ticket";

describe("ticket", () => {
  describe("readTicket", () => {
    it("should send an email that has an account to the login page", () => {
      expect(
        readTicket(
          new URLSearchParams({ __clerk_ticket: "tkt", __clerk_status: "sign_in" }),
        ),
      ).toEqual({ token: "tkt", page: "/login" });
    });

    it("should send an email without an account to the sign-up page", () => {
      expect(
        readTicket(
          new URLSearchParams({ __clerk_ticket: "tkt", __clerk_status: "sign_up" }),
        ),
      ).toEqual({ token: "tkt", page: "/sign-up" });
    });

    it("should read nothing once Clerk accepted the invitation", () => {
      expect(
        readTicket(
          new URLSearchParams({ __clerk_ticket: "tkt", __clerk_status: "complete" }),
        ),
      ).toBeNull();
    });

    it("should read nothing from a link without a ticket", () => {
      expect(readTicket(new URLSearchParams())).toBeNull();
      expect(readTicket(new URLSearchParams({ __clerk_status: "sign_up" }))).toBeNull();
      expect(
        readTicket(
          new URLSearchParams({ __clerk_ticket: "", __clerk_status: "sign_up" }),
        ),
      ).toBeNull();
    });

    it("should read nothing for a status Clerk does not send", () => {
      expect(readTicket(new URLSearchParams({ __clerk_ticket: "tkt" }))).toBeNull();
      expect(
        readTicket(
          new URLSearchParams({ __clerk_ticket: "tkt", __clerk_status: "nope" }),
        ),
      ).toBeNull();
    });
  });
});
