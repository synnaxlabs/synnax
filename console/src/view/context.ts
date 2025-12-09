// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type view } from "@synnaxlabs/client";
import { context, type Flux } from "@synnaxlabs/pluto";

export interface FormContextValue {
  search: (term: string) => void;
  save: (opts?: Flux.FetchOptions) => void;
}

export const [FormContext, useFormContext] = context.create<FormContextValue>({
  displayName: "View.FormContext",
  providerName: "View.Form",
});

export interface View extends view.View {
  static?: true;
}

export interface StaticView extends View {
  static: true;
}

export interface ContextValue {
  resourceType: ontology.ResourceType;
  selected: view.Key;
  select: (key: view.Key) => void;
  editable: boolean;
  getView: (key: view.Key) => view.View | undefined;
  staticViews: Set<view.Key>;
}

export const [Context, useContext] = context.create<ContextValue>({
  displayName: "View.Context",
  providerName: "View.Provider",
});
