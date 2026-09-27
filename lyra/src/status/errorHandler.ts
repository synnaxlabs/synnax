// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors } from "@synnaxlabs/x";
import type z from "zod";

import { type Crude, fromException, type Status, toString } from "@/status/status";

/** Adds a status to the enclosing aggregator. */
export interface Adder {
  <Details extends z.ZodType = z.ZodNever>(spec: Crude<Details>): void;
}

export interface ErrorHandler {
  /** Reports the given error, or runs the given function and reports a rejection. */
  (
    funcOrExc: unknown,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): void;
}

export interface AsyncErrorHandler {
  /** Reports the given error, or runs the given function and reports a rejection. */
  (
    funcOrExc: unknown,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
}

type Skip = errors.Matchable | errors.Matchable[] | undefined;

const checkSkip = (err: unknown, skip: Skip): boolean => {
  if (Array.isArray(skip)) return skip.some((matcher) => matcher.matches(err));
  return skip?.matches(err) ?? false;
};

const parseException = (exc: unknown, message?: string, skip?: Skip): Status | null => {
  const stat = fromException(exc, message);
  console.error(toString(stat));
  if (checkSkip(exc, skip)) return null;
  return stat;
};

const handleException = <ExcOrFunc>(
  excOrFunc: ExcOrFunc,
  add: Adder,
  message?: string,
  skip?: Skip,
): excOrFunc is ExcOrFunc & (() => Promise<void> | void) => {
  if (typeof excOrFunc === "function") return true;
  const stat = parseException(excOrFunc, message, skip);
  if (stat != null) add(stat);
  return false;
};

const handleFunc = async (
  func: () => Promise<void> | void,
  add: Adder,
  message?: string,
  skip?: Skip,
): Promise<void> => {
  try {
    const promise = func();
    // Skip the added microtask if the function returns void instead of a promise.
    if (promise != null) await promise;
  } catch (exc) {
    const stat = parseException(exc, message, skip);
    if (stat != null) add(stat);
  }
};

export const createErrorHandler =
  (add: Adder): ErrorHandler =>
  (excOrFunc, message, skip): void => {
    if (!handleException(excOrFunc, add, message, skip)) return;
    void handleFunc(excOrFunc, add, message, skip);
  };

export const createAsyncErrorHandler =
  (add: Adder): AsyncErrorHandler =>
  async (func, message, skip): Promise<void> => {
    if (!handleException(func, add, message, skip)) return;
    await handleFunc(func, add, message, skip);
  };
