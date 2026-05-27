// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { type aether } from "@/aether/aether";
import {
  buildProviderStack,
  type MountedProviders,
  type StackOptions,
} from "@/aether/test/mount";

export interface RenderAetherOptions extends StackOptions {
  /** Standard React Testing Library render options (minus `wrapper`, which we own). */
  rtl?: Omit<RenderOptions, "wrapper">;
}

/** RTL render result plus references into the worker tree set up by the harness. */
export interface RenderAetherResult extends RenderResult {
  /** Root of the worker tree. Useful for `root.findChildAtPath(...)` to grab any
   * mounted component by path. */
  root: aether.Root;
  /** Provider instances mounted on the worker, by name. */
  providers: MountedProviders;
}

/**
 * Render a React component with a full Aether stack already mounted on the worker
 * side, so the component's `Aether.use` calls register under telem (and the optional
 * render provider) without the test having to scaffold a multi-level provider tree by
 * hand.
 *
 * Use this for tests that exercise the React + worker boundary — clicks, form input,
 * dispatched actions, etc. For pure worker-side renderer testing, prefer
 * `aetherTest.mount` directly.
 */
export const renderAether = (
  ui: ReactElement,
  options: RenderAetherOptions = {},
): RenderAetherResult => {
  const { rtl, ...stackOptions } = options;
  const stack = buildProviderStack(stackOptions);

  const Wrapper: FC<PropsWithChildren> = ({ children }) => (
    <Aether.Provider worker={stack.mainSide}>
      <Aether.Composite path={stack.basePath}>{children}</Aether.Composite>
    </Aether.Provider>
  );

  const result = render(ui, { ...(rtl ?? {}), wrapper: Wrapper });
  return { ...result, root: stack.root, providers: stack.providers };
};
