// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type secret, type Synnax } from "@synnaxlabs/client";

import { uniqueName } from "@/testutil";

/** Creates a stored secret with a unique name and adds its key to created. */
export const createSecret = async (
  client: Synnax,
  created: secret.Key[],
): Promise<secret.Secret> => {
  const s = await client.secrets.set({
    name: uniqueName("secret"),
    source: "stored",
    value: "hunter2",
  });
  created.push(s.key);
  return s;
};
