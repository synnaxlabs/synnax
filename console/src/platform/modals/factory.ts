// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Session } from "@/session";

/**
 * A typed fire-and-forget opener: opens a modal and returns immediately. Its params
 * argument is omittable when every field is optional and required otherwise.
 */
export type Opener<Params> = optional.Arg<Params, void>;

/**
 * A typed prompt opener: opens a modal and resolves with its result, or null. Its params
 * argument is omittable when every field is optional and required otherwise.
 */
export type Prompt<Result, Params> = optional.Arg<Params, Promise<Result | null>>;

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
export const create =
  <Params extends Session.Modals.ReservedParams = Record<never, never>>(
    Component: Session.Modals.Content<Params, void>,
  ): OpenHook<Params> =>
  (): Opener<Params> => {
    const store = Session.Modals.useStore("Modals.create");
    return useCallback(
      (params?: Params) => store.push(Component, params, () => {}),
      [store],
    );
  };

/**
 * prompt defines a result-returning modal. It returns the prompt hook directly; the
 * hook's function takes the modal's typed params and resolves with the renderer's result
 * (or null on dismissal).
 */
export const createPrompt =
  <Result, Params extends Session.Modals.ReservedParams = Record<never, never>>(
    Component: Session.Modals.Content<Params, Result>,
  ): PromptHook<Params, Result> =>
  (): Prompt<Result, Params> => {
    const store = Session.Modals.useStore("Modals.createPrompt");
    return useCallback(
      (params?: Params) =>
        new Promise<Result | null>((resolve) => store.push(Component, params, resolve)),
      [store],
    );
  };
