// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";

export interface FormContextValue {
  answered: boolean;
}

export const [FormContext, useFormContext] = context.create<FormContextValue>({
  displayName: "View.FormContext",
  providerName: "View.Form",
});

export type View = view.View & { static?: true };

export type StaticView = View & { static: true };

export interface ContextValue {
  resourceType: view.Type;
  selected: view.Key;
  select: (key: view.Key) => void;
  /** True while the toolbar is open: the view is always editable, or editing is on. */
  editing: boolean;
  /** True while editing a view the user may update. Only then can parts write it. */
  editable: boolean;
  staticViews: view.Key[];
  getInitialView: () => view.View;
  /** True when the search leads the views bar, over a filter bar. */
  searchInViews: boolean;
  /** The term the lists of every view are searched by. */
  searchTerm: string;
  setSearchTerm: (term: string) => void;
}

export const [Context, useContext] = context.create<ContextValue>({
  displayName: "View.Context",
  providerName: "View.Frame",
});
