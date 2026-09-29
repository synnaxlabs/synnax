// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";

import { uniqueName } from "@/testutil";

export const client = createTestClient();

/** A CSV import that holds one message, Engine, with one field, Rpm. */
export const ENGINE_CSV = "message,field,start_bit,bit_length\nEngine,Rpm,0,16";

/** Creates a library on the Core with a unique name and the given entries. */
export const createLibrary = async (
  entries: library.Entry[] = [],
  c: Client = client,
): Promise<library.Library> =>
  await c.libraries.create({ name: uniqueName("library"), entries });
