// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type ComponentType } from "react";

/**
 * The props passed to a modal {@link Renderer}. The renderer reads its arguments from
 * the active modal via {@link useArgs} and signals completion through onClose.
 */
export interface RendererProps {
  /** The unique key of the modal. */
  layoutKey: string;
  /**
   * Closes the modal, optionally resolving the open promise with the given result.
   * Called with no argument (or null) when the modal is dismissed without a result.
   */
  onClose: (result?: unknown) => void;
  visible: boolean;
  focused: boolean;
  active: boolean;
}

/** A React component that renders a modal of a particular registered type. */
export interface Renderer extends ComponentType<RendererProps> {}

export interface Renderers extends Record<string, Renderer> {}

const [RendererContext, useRendererContext] = context.create<Renderers>({
  defaultValue: {},
  displayName: "Modals.RendererContext",
});

/** Provides the type-to-component registry consumed by {@link useRenderer}. */
export const RendererProvider = RendererContext;

/** Resolves the registered {@link Renderer} for the given modal type. */
export const useRenderer = (type: string): Renderer => {
  const r = useRendererContext()[type];
  if (r == null) throw new Error(`no renderer for modal type ${type}`);
  return r;
};
