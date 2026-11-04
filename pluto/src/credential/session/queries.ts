// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth, type ontology } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export type RetrieveQuery = {
  subject: ontology.ID;
};

export const { useResult, useInvalidate } = Flux.createRetrieve<
  RetrieveQuery,
  auth.Session[]
>({
  name: "sessions",
  retrieve: async ({ client, query: { subject } }) =>
    await client.auth.sessions.retrieve([subject]),
});
