// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Ticket is the team invitation that Clerk adds to the link in an invitation email. */
export interface Ticket {
  token: string;
  /**
   * page is the auth page that accepts the invitation: login for an email that has an
   * account, sign-up for one that does not.
   */
  page: "/login" | "/sign-up";
}

/**
 * readTicket reads the invitation from an invitation link. Returns null when the link
 * carries none, or when Clerk already accepted it.
 */
export const readTicket = (params: URLSearchParams): Ticket | null => {
  const token = params.get("__clerk_ticket") ?? "";
  if (token === "") return null;
  switch (params.get("__clerk_status")) {
    case "sign_in":
      return { token, page: "/login" };
    case "sign_up":
      return { token, page: "/sign-up" };
    default:
      return null;
  }
};
