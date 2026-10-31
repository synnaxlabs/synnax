// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Mode is how a site asks about cookies, or the Desktop login about usage data: `ask`
 * for consent, or `notice` that they are on.
 */
export type Mode = "ask" | "notice";

/**
 * ASK lists the countries that get `ask`: the EU, the rest of the EEA, the UK, and
 * Switzerland. Some location databases send the reserved codes EU and UK for them.
 */
const ASK: ReadonlySet<string> = new Set(
  [
    "AT BE BG CY CZ DE DK EE ES FI FR GR HR HU IE IT LT LU LV MT NL PL PT RO SE SI SK",
    "IS LI NO",
    "GB CH",
    "EU UK",
  ].flatMap((codes) => codes.split(" ")),
);

/** COUNTRY is the shape of an ISO 3166-1 alpha-2 country code. */
const COUNTRY = /^[A-Z]{2}$/;

/** USER_ASSIGNED are the codes ISO 3166-1 leaves free, which name no country. */
const USER_ASSIGNED = /^(AA|Q[M-Z]|X[A-Z]|ZZ)$/;

/**
 * modeFor picks the consent mode for the `x-vercel-ip-country` of a request. A missing
 * or unknown country gets `ask`.
 */
export const modeFor = (country: string | null): Mode => {
  if (country == null || !COUNTRY.test(country) || USER_ASSIGNED.test(country))
    return "ask";
  return ASK.has(country) ? "ask" : "notice";
};
