// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Creates the sample project on a Core and starts its Arc programs, for a Console that
// cannot create it itself, such as the web Console.
//
// Usage: node scripts/create_sample.ts [PORT | URL], such as 9090 (localhost) or
// https://core.example.com. The default is 9090.

import { Synnax } from "@synnaxlabs/client";

import { create } from "../src/feature/project/sample.ts";
import { RECENT_KEY } from "../src/session/range/slice.ts";
import { parse } from "./coreURL.ts";

const client = new Synnax({
  ...parse(process.argv[2] ?? "9090"),
  username: "synnax",
  password: "seldon",
});
await create(client, RECENT_KEY);
await client.close();
