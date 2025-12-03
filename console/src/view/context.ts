// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { context, type Flux, type List, type state } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

export interface Request extends List.PagerParams, record.Unknown {}

export interface ContextValue<R extends Request = Request> {
  request: R;
  onRequestChange: (setter: state.SetArg<R>, opts?: Flux.AsyncListOptions) => void;
  editable: boolean;
  setEditable: state.Setter<boolean>;
  visible: boolean;
  resourceType: ontology.ResourceType;
}

const [Provider_, useContext_] = context.create<ContextValue>({
  displayName: "Frame.Context",
  providerName: "Frame.Provider",
});

export const Provider = Provider_;

export const useContext = <R extends Request = Request>(
  componentName: string,
): ContextValue<R> => useContext_(componentName) as unknown as ContextValue<R>;
