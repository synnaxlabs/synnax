// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * @returns the host and port as one address. An IPv6 host gets brackets, which keep its
 * colons apart from the colon before the port.
 */
export const formatAddress = (host: string, port: number): string =>
  `${host.includes(":") && !host.startsWith("[") ? `[${host}]` : host}:${port}`;
