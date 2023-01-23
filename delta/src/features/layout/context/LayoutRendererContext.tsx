// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, useContext } from "react";

import { LayoutRenderer } from "../types";

export type LayoutRenderers = Record<string, LayoutRenderer>;

const LayoutRendererContext = createContext<LayoutRenderers>({});

export const LayoutRendererProvider = LayoutRendererContext.Provider;

export const useLayoutRenderer = (type: string): LayoutRenderer => {
  const r = useContext(LayoutRendererContext)[type];
  if (r == null) throw new Error(`no renderer for layout type ${type}`);
  return r;
};
