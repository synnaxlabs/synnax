// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export const { useUpdate: useSet } = Flux.createUpdate<auth.password.SetParams>({
  name: "password",
  verbs: verbs.SET,
  update: async ({ client, data }) => {
    await client.auth.passwords.set(data);
    return data;
  },
});
