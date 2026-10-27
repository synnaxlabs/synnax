// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  act,
  render as rtlRender,
  renderHook as rtlRenderHook,
  type RenderHookOptions as RTLRenderHookOptions,
  type RenderHookResult as RTLRenderHookResult,
  type RenderOptions as RTLRenderOptions,
  type RenderResult as RTLRenderResult,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useEffect } from "react";

import { Aether } from "@/aether";
import { type aether } from "@/aether/aether";
import {
  buildStack,
  type MountedProviders,
  type ProviderOptions,
} from "@/testutil/internal";
import { type canvasTest } from "@/vis/render/testutil";

/** Options for {@link render}: provider toggles plus standard RTL render options
 * (minus `wrapper`, which the harness owns). */
export interface RenderOptions extends ProviderOptions {
  rtl?: Omit<RTLRenderOptions, "wrapper">;
}

interface Mounted {
  /** Root of the worker tree. Use `root.findChildAtPath(...)` to grab any mounted
   * component by path. */
  root: aether.Root;
  /** Provider instances mounted on the worker, by name. */
  providers: MountedProviders;
  /** The render recorder, if `render` was enabled; otherwise `null`. */
  recorder: canvasTest.Recorder | null;
}

/** RTL render result plus references into the worker tree the harness mounted. */
export interface RenderResult extends RTLRenderResult, Mounted {}

/** Options for {@link renderHook}: provider toggles plus standard RTL renderHook
 * options (minus `wrapper`, which the harness owns). */
export interface RenderHookOptions<Props> extends ProviderOptions {
  rtl?: Omit<RTLRenderHookOptions<Props>, "wrapper">;
}

/** RTL renderHook result plus references into the worker tree the harness mounted. */
export interface RenderHookResult<Result, Props>
  extends RTLRenderHookResult<Result, Props>, Mounted {}

const mount = (options: ProviderOptions): [FC<PropsWithChildren>, Mounted] => {
  const stack = buildStack(options);
  const Wrapper: FC<PropsWithChildren> = ({ children }) => (
    <Aether.Provider worker={stack.driver.mainSide}>
      <Aether.Composite path={stack.basePath}>{children}</Aether.Composite>
    </Aether.Provider>
  );
  const { driver, providers, recorder } = stack;
  return [Wrapper, { root: driver.root, providers, recorder }];
};

/**
 * Renders a hook that suspends on a cold cache, resolving once its render commits.
 * RTL's own `renderHook` never commits a tree that suspends during the initial render,
 * leaving `result.current` null forever. Never call this inside an `act` scope: the
 * commit it waits on cannot land until that scope exits.
 */
export const renderHookSuspended = async <Result, Props>(
  useHook: (props: Props) => Result,
  options?: RTLRenderHookOptions<Props>,
): Promise<RTLRenderHookResult<Result, Props>> => {
  let committed = false;
  const Tracked = (props: Props): Result => {
    const result = useHook(props);
    useEffect(() => {
      committed = true;
    }, []);
    return result;
  };
  let rendered!: RTLRenderHookResult<Result, Props>;
  // An async act drains microtasks and React work, but not a fetch still in flight, so
  // the commit is waited on rather than assumed.
  await act(async () => {
    rendered = rtlRenderHook(Tracked, options);
  });
  await waitFor(() => {
    if (!committed) throw new Error("the suspended render never committed");
  });
  return rendered;
};

/**
 * Render a React component with a Synnax provider stack already mounted on the worker
 * side, so the component's Aether hooks register under the stack without the test
 * scaffolding a multi-level provider tree by hand. Use this for tests that exercise the
 * React + worker boundary (clicks, form input, dispatched actions). For pure
 * worker-side renderer testing, use `renderAether`.
 */
export const render = (ui: ReactElement, options: RenderOptions = {}): RenderResult => {
  const { rtl, ...providerOptions } = options;
  const [Wrapper, mounted] = mount(providerOptions);
  return { ...rtlRender(ui, { ...(rtl ?? {}), wrapper: Wrapper }), ...mounted };
};

/** Renders a hook under the provider stack that {@link render} mounts. */
export const renderHook = <Result, Props>(
  useHook: (props: Props) => Result,
  options: RenderHookOptions<Props> = {},
): RenderHookResult<Result, Props> => {
  const { rtl, ...providerOptions } = options;
  const [Wrapper, mounted] = mount(providerOptions);
  return {
    ...rtlRenderHook(useHook, { ...(rtl ?? {}), wrapper: Wrapper }),
    ...mounted,
  };
};
