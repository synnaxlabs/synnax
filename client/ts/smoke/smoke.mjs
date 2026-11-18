// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";

// Browsers have no process, so browser.mjs sets smokeEnv.
const env = globalThis.process?.env ?? globalThis.smokeEnv;
const client = new Synnax({
  host: env.SYNNAX_TEST_HOST ?? "localhost",
  port: Number(env.SYNNAX_TEST_PORT ?? 9090),
  username: "synnax",
  password: "seldon",
  secure: false,
});
const channels = await client.channels.retrieve({ limit: 3 });
if (channels.length === 0) throw new Error("The Core has no channels");
const streamer = await client.openStreamer(channels.map((ch) => ch.key));
streamer.close();
await client.close();
globalThis.smoked = true;
