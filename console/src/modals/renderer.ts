// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type FC } from "react";

export interface RendererProps {
  layoutKey: string;
  onClose: (result?: unknown) => void;
}

export interface Renderer extends FC<RendererProps> {}

export interface Renderers extends Record<string, Renderer> {}

const [RendererContext, useRendererContext] = context.create<Renderers>({
  defaultValue: {},
  displayName: "Modals.RendererContext",
});

export { RendererContext };

export const useRenderer = (type: string): Renderer => {
  const r = useRendererContext()[type];
  if (r == null) throw new Error(`no renderer for modal type ${type}`);
  return r;
};
