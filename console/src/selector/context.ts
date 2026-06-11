// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, use } from "react";

import { type Selectable } from "@/selector/Selector";

// Context carries the app-wide selectable registry, assembled once in Console.tsx
// from each module's SELECTABLES (mirroring how LAYOUT_RENDERERS is assembled and
// injected). The value must be a module-level constant: useVisible maps hooks over
// it, so its length and order may never change between renders.
const Context = createContext<Selectable[]>([]);
Context.displayName = "Selector.Selectables";

export const Provider = Context.Provider;

// useSelectables returns the app-wide selectable registry.
export const useSelectables = (): Selectable[] => use(Context);

// useVisible reports whether at least one registered selectable is visible to the
// current user — i.e. whether "create a component" affordances should show.
export const useVisible = (): boolean =>
  useSelectables()
    .map((s) => s.useVisible?.() ?? true)
    .some(Boolean);
