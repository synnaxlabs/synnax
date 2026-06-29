// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { useStore } from "@/layered/session/modals/Provider";
import { type RenderProps } from "@/layered/session/modals/store";

/**
 * The opener's parameter list for a given params type: empty when the modal takes no
 * params, optional when every field is optional, and required otherwise.
 */
export type ParamList<Params> = void extends Params
  ? []
  : {} extends Params
    ? [params?: Params]
    : [params: Params];

/** A typed fire-and-forget opener: opens a modal and returns immediately. */
export interface Opener<Params> {
  (...params: ParamList<Params>): void;
}

/** A typed prompt opener: opens a modal and resolves with its result, or null. */
export interface Prompt<Result, Params> {
  (...params: ParamList<Params>): Promise<Result | null>;
}

/**
 * A fire-and-forget modal. Calling the hook returns an opener that pushes the modal and
 * returns immediately.
 */
export interface OpenHook<Params> {
  (): Opener<Params>;
}

/**
 * A result-returning modal. Calling the hook returns an opener that resolves with the
 * value the renderer passes to close, or null on dismissal.
 */
export interface PromptHook<Params, Result> {
  (): Prompt<Result, Params>;
}

/**
 * create defines a fire-and-forget modal (a form that performs its own side effects and
 * dismisses itself). It returns the open hook directly; the hook's function takes the
 * modal's typed params.
 */
export const create = <Params = void>(
  Component: FC<RenderProps<Params, void>>,
): OpenHook<Params> => {
  const useOpen = (): Opener<Params> => {
    const store = useStore();
    return useCallback(
      (...params: ParamList<Params>) =>
        store.push({
          key: id.create(),
          render: Component,
          params: (params[0] ?? {}) as Params,
          resolve: () => {},
        }),
      [store],
    );
  };
  return useOpen;
};

/**
 * prompt defines a result-returning modal. It returns the prompt hook directly; the
 * hook's function takes the modal's typed params and resolves with the renderer's result
 * (or null on dismissal).
 */
export const prompt = <Result, Params = void>(
  Component: FC<RenderProps<Params, Result>>,
): PromptHook<Params, Result> => {
  const usePrompt = (): Prompt<Result, Params> => {
    const store = useStore();
    return useCallback(
      (...params: ParamList<Params>) =>
        new Promise<Result | null>((resolve) =>
          store.push({
            key: id.create(),
            render: Component,
            params: (params[0] ?? {}) as Params,
            resolve,
          }),
        ),
      [store],
    );
  };
  return usePrompt;
};
