// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { TimeStamp } from "@synnaxlabs/x";

const EDITIONS: Record<string, string> = { d: "Desktop", e: "Enterprise" };

/** The display name of a license edition code. */
export const editionLabel = ({ ed }: license.License): string => EDITIONS[ed] ?? ed;

/** The license's term on one line: when it ends, or the versions it covers. */
export const describeTerm = ({ exp, mv }: license.License): string => {
  const until = exp == null ? null : TimeStamp.seconds(exp).toString("ISODate");
  if (until == null && mv == null) return "Perpetual";
  if (until == null) return `Perpetual, covers versions up to ${mv}`;
  if (mv == null) return `Expires ${until}`;
  return `Subscription until ${until}, then versions up to ${mv}`;
};

/** The channel cap on one line. */
export const describeChannels = ({ ch }: license.License): string =>
  ch === 0 ? "Unlimited" : `Up to ${ch}`;

/** Joins host hashes the way the portal's activation page reads them. */
export const joinFingerprint = (fingerprint: string[]): string =>
  fingerprint.join(", ");
