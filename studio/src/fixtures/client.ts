// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";

export interface ConnectionOptions {
  host?: string;
  port?: number;
  username?: string;
  password?: string;
}

/** defaultPort resolves the port of the capture's core, set by the pipeline. */
export const defaultPort = (): number => Number(process.env.SYNNAX_STUDIO_PORT ?? 9090);

/** connect opens a client against the capture's core, using its default credentials. */
export const connect = ({
  host = "localhost",
  port = defaultPort(),
  username = "synnax",
  password = "seldon",
}: ConnectionOptions = {}): Synnax => new Synnax({ host, port, username, password });
