// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export type RetrieveQuery = Record<string, never>;

export const { use, useResult } = Flux.createRetrieve<RetrieveQuery, license.Info>({
  name: "license",
  retrieve: async ({ client }) => await client.license.retrieve(),
  onChange: ({ client }, handler) => client.license.onChange(handler),
  getCached: ({ client }) => client.license.getCached(),
});
