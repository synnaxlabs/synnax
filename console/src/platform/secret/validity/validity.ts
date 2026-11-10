// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type secret } from "@synnaxlabs/client";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";

const EXPIRY_WARNING = TimeSpan.days(30);

const createFormat = (timeZone?: string) =>
  new Intl.DateTimeFormat("en-US", {
    month: "long",
    day: "numeric",
    year: "numeric",
    timeZone,
  });

const LOCAL = createFormat();

// Certificate dates read in UTC, as the expiry statuses of the Core do.
const UTC = createFormat("UTC");

/**
 * Describes where the value of a secret comes from: "Set on May 4, 2026", or "Read from
 * token" for a file secret.
 */
export const describeSecret = (s: secret.Secret): string =>
  s.source === "file"
    ? `Read from ${s.fileName}`
    : `Set on ${LOCAL.format(s.updated.date())}`;

export interface Validity {
  text: string;
  status?: "warning" | "error";
}

const describeDetails = (
  { notBefore, notAfter }: certificate.Details,
  now: TimeStamp,
): Validity => {
  const formatDate = (ts: TimeStamp) => UTC.format(ts.date());
  if (now.before(notBefore))
    return { text: `Valid from ${formatDate(notBefore)}`, status: "warning" };
  const expiry = formatDate(notAfter);
  if (now.afterEq(notAfter)) return { text: `Expired ${expiry}`, status: "error" };
  if (now.afterEq(notAfter.sub(EXPIRY_WARNING)))
    return { text: `Expires ${expiry}`, status: "warning" };
  return { text: `Expires ${expiry}` };
};

/** Describes whether a certificate is valid at the given time. */
export const describeCertificate = (
  c: certificate.Certificate,
  now: TimeStamp,
): Validity => {
  if (c.source === "stored") return describeDetails(c.details, now);
  if (c.details == null) return { text: c.certificatePath };
  return describeDetails(c.details, now);
};
