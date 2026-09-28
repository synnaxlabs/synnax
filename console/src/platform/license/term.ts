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
export const editionLabel = ({ edition }: license.License): string =>
  EDITIONS[edition] ?? edition;

/** The license's term on one line: when it ends, or the versions it covers. */
export const describeTerm = ({ exp, maxVersion }: license.License): string => {
  const until = exp == null ? null : TimeStamp.seconds(exp).toString("ISODate");
  if (until == null && maxVersion == null) return "Perpetual";
  if (until == null) return `Perpetual, covers versions up to ${maxVersion}`;
  if (maxVersion == null) return `Expires ${until}`;
  return `Subscription until ${until}, then versions up to ${maxVersion}`;
};

/** The channel cap on one line. */
export const describeChannels = ({ channels }: license.License): string =>
  channels === 0 ? "Unlimited" : `Up to ${channels}`;
