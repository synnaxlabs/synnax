// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** STATE is the shape of the one-time value the Desktop app mints for a sign-in. */
export const STATE = /^[A-Za-z0-9_-]{16,128}$/;

/** SCHEME is the URL scheme the Desktop app registers. */
export const SCHEME = "synnax-desktop";

export interface Linked {
  token: string;
  secret: string;
  activation: string;
  email: string;
}

/** activateURL builds the link that hands a linked machine its token. */
export const activateURL = (state: string, linked: Linked): string => {
  const params = new URLSearchParams({
    state,
    token: linked.token,
    secret: linked.secret,
    activation: linked.activation,
    email: linked.email,
  });
  return `${SCHEME}://activate?${params.toString()}`;
};
