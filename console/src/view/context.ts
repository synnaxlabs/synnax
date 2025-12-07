// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { context, type Flux } from "@synnaxlabs/pluto";

export interface ContextValue {
  editable: boolean;
  resourceType: ontology.ResourceType;
  search: (term: string) => void;
  save: (opts?: Flux.FetchOptions) => void;
}

export const [Provider, useContext] = context.create<ContextValue>({
  displayName: "View.Context",
  providerName: "View.Frame",
});
