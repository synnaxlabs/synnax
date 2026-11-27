// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface Core {
  host: string;
  port: number;
  insecure: boolean;
}

/**
 * Parses a Core address: a bare port on localhost, or an `http` or `https` URL. A URL
 * with no port uses 80 for `http` and 443 for `https`.
 *
 * @throws Error when the address is neither a port nor an `http` or `https` URL.
 */
export const parse = (address: string): Core => {
  if (/^\d+$/.test(address))
    return { host: "localhost", port: Number(address), insecure: true };
  const url = new URL(address);
  if (url.protocol !== "http:" && url.protocol !== "https:")
    throw new Error(`${address} is not an http or https URL`);
  const insecure = url.protocol !== "https:";
  const port = url.port === "" ? (insecure ? 80 : 443) : Number(url.port);
  return { host: url.hostname, port, insecure };
};
