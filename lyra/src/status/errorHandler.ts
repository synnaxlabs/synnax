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

import { type Crude, fromException, toString } from "@/status/status";

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

/** Prints a reported error. */
export interface Log {
  (message: string): void;
}

type Skip = errors.Matchable | errors.Matchable[] | undefined;

type Report = (exc: unknown, message?: string, skip?: Skip) => void;

const checkSkip = (err: unknown, skip: Skip): boolean => {
  if (Array.isArray(skip)) return skip.some((matcher) => matcher.matches(err));
  return skip?.matches(err) ?? false;
};

const createReport =
  (add: Adder, log: Log): Report =>
  (exc, message, skip) => {
    if (checkSkip(exc, skip)) return;
    const stat = fromException(exc, message);
    log(toString(stat));
    add(stat);
  };

const isFunc = (v: unknown): v is () => Promise<void> | void => typeof v === "function";

const run = async (
  func: () => Promise<void> | void,
  report: Report,
  message?: string,
  skip?: Skip,
): Promise<void> => {
  try {
    const promise = func();
    // Skip the added microtask if the function returns void instead of a promise.
    if (promise != null) await promise;
  } catch (exc) {
    report(exc, message, skip);
  }
};

export const createErrorHandler = (add: Adder, log: Log): ErrorHandler => {
  const report = createReport(add, log);
  return (excOrFunc, message, skip) => {
    if (isFunc(excOrFunc)) void run(excOrFunc, report, message, skip);
    else report(excOrFunc, message, skip);
  };
};

export const createAsyncErrorHandler = (add: Adder, log: Log): AsyncErrorHandler => {
  const report = createReport(add, log);
  return async (excOrFunc, message, skip) => {
    if (isFunc(excOrFunc)) await run(excOrFunc, report, message, skip);
    else report(excOrFunc, message, skip);
  };
};
