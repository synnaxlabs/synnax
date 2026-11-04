// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/client";

import { type ConnectionConfig } from "@/feature/opcua/device/types";

const MARGIN = TimeSpan.seconds(2);

/**
 * @returns how long to wait for a scan command that connects to the server, which can
 * wait the client timeout once to connect and once more for its request.
 */
export const commandTimeout = ({ clientTimeoutMs }: ConnectionConfig): TimeSpan =>
  TimeSpan.milliseconds(2 * clientTimeoutMs).add(MARGIN);
