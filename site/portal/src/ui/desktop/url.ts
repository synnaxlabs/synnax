// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { parse } from "@/server/license/fingerprint";
import { MAX_NAME_LENGTH } from "@/server/license/limits";

/** STATE is the shape of the one-time value the Desktop app mints for a login. */
export const STATE = /^[A-Za-z0-9_-]{16,128}$/;

/** SCHEME is the URL scheme the Desktop app registers. */
export const SCHEME = "synnax-desktop";

export interface Linked {
  key: string;
  secret: string;
  activation: string;
  email: string;
}

/** activateURL builds the link that hands a linked machine its license key. */
export const activateURL = (state: string, linked: Linked): string => {
  const params = new URLSearchParams({
    state,
    key: linked.key,
    secret: linked.secret,
    activation: linked.activation,
    email: linked.email,
  });
  return `${SCHEME}://activate?${params.toString()}`;
};

export interface Query {
  state: string;
  fingerprint: string[];
  /** name is the machine name, trimmed to its bound. */
  name: string;
  /** problem is why the page cannot link, when the query did not come from the app. */
  problem: string | null;
}

const NOT_FROM_APP = "This link did not come from the Synnax Desktop app.";

const readFingerprint = (text: string): string[] | null => {
  try {
    return parse(text);
  } catch {
    return null;
  }
};

/** read validates the query the Desktop app opens the login page with. */
export const read = (params: URLSearchParams): Query => {
  const state = params.get("state") ?? "";
  const name = (params.get("name") ?? "").trim().slice(0, MAX_NAME_LENGTH);
  const fingerprint = readFingerprint(params.get("fp") ?? "");
  const valid = STATE.test(state) && name !== "" && fingerprint != null;
  return {
    state,
    name,
    fingerprint: fingerprint ?? [],
    problem: valid ? null : NOT_FROM_APP,
  };
};
