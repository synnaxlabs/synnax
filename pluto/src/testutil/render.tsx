// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  render as rtlRender,
  type RenderOptions as RTLRenderOptions,
  type RenderResult as RTLRenderResult,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { type aether } from "@/aether/aether";
import {
  buildStack,
  type MountedProviders,
  type ProviderOptions,
} from "@/testutil/providers";
import { type canvasTest } from "@/vis/render/test";

/** Options for {@link render}: provider toggles plus standard RTL render options
 * (minus `wrapper`, which the harness owns). */
export interface RenderOptions extends ProviderOptions {
  rtl?: Omit<RTLRenderOptions, "wrapper">;
}

/** RTL render result plus references into the worker tree the harness mounted. */
export interface RenderResult extends RTLRenderResult {
  /** Root of the worker tree. Use `root.findChildAtPath(...)` to grab any mounted
   * component by path. */
  root: aether.Root;
  /** Provider instances mounted on the worker, by name. */
  providers: MountedProviders;
  /** The render recorder, if `render` was enabled; otherwise `null`. */
  recorder: canvasTest.Recorder | null;
}

/**
 * Render a React component with a Synnax provider stack already mounted on the worker
 * side, so the component's `Aether.use` calls register under the stack without the test
 * scaffolding a multi-level provider tree by hand.
 *
 * Use this for tests that exercise the React + worker boundary (clicks, form input,
 * dispatched actions). For pure worker-side renderer testing, use `renderAether`.
 */
export const render = (ui: ReactElement, options: RenderOptions = {}): RenderResult => {
  const { rtl, ...providerOptions } = options;
  const stack = buildStack(providerOptions);

  const Wrapper: FC<PropsWithChildren> = ({ children }) => (
    <Aether.Provider worker={stack.driver.mainSide}>
      <Aether.Composite path={stack.basePath}>{children}</Aether.Composite>
    </Aether.Provider>
  );

  const result = rtlRender(ui, { ...(rtl ?? {}), wrapper: Wrapper });
  return {
    ...result,
    root: stack.driver.root,
    providers: stack.providers,
    recorder: stack.recorder,
  };
};
