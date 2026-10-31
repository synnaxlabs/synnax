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

import { readFileSync } from "node:fs";

import { Synnax } from "@synnaxlabs/client";

import { create } from "../src/feature/project/sample.ts";
import { parse } from "./coreURL.ts";

const client = new Synnax({
  ...parse(process.argv[2] ?? "9090"),
  username: "synnax",
  password: "seldon",
});
const read = (name: string): string =>
  readFileSync(new URL(`../src/feature/project/${name}`, import.meta.url), "utf8");
await create(client, {
  simulator: read("sampleSimulator.arc"),
  sequence: read("sampleSequence.arc"),
});
await client.close();
