// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { Flux, Pluto, Status, Synnax } from "@synnaxlabs/pluto";
import { type errors, narrow, status } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

interface Adder {
  <Details = never>(spec: status.Crude<Details>): void;
}

interface ErrorHandler {
  (
    func: () => Promise<void> | void,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): void;
  (exc: unknown, message?: string, skip?: errors.Matchable | errors.Matchable[]): void;
}

interface AsyncErrorHandler {
  (
    func: () => Promise<void> | void,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
  (
    exc: unknown,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
}

const checkSkip = (
  err: unknown,
  skip: errors.Matchable | errors.Matchable[] | undefined,
): boolean => {
  if (Array.isArray(skip)) return skip.some((matcher) => matcher.matches(err));
  return skip?.matches(err) ?? false;
};

const formatError = (stat: status.Status): void => {
  const parts: string[] = [`${stat.variant.toUpperCase()}: ${stat.message}`];
  if (stat.description)
    try {
      const parsed = JSON.parse(stat.description);
      parts.push(`Description:\n${JSON.stringify(parsed, null, 2)}`);
    } catch {
      parts.push(`Description: ${stat.description}`);
    }
  if ("details" in stat && narrow.isObject(stat.details) && "stack" in stat.details)
    parts.push(`Stack Trace:\n${String(stat.details.stack)}`);
  console.error(parts.join("\n\n"));
};

const parseException = (
  exc: unknown,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): status.Status | null => {
  const stat = status.fromException(exc, message);
  formatError(stat);
  if (checkSkip(exc, skip)) return null;
  return stat;
};

const handleException = <ExcOrFunc,>(
  excOrFunc: ExcOrFunc,
  add: Adder,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): excOrFunc is ExcOrFunc & (() => Promise<void> | void) => {
  if (typeof excOrFunc === "function") return true;
  const stat = parseException(excOrFunc, message, skip);
  if (stat != null) add(stat);
  return false;
};

const handleFunc = async <Func extends () => Promise<void> | void>(
  func: Func,
  add: Adder,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): Promise<void> => {
  try {
    const promise = func();
    if (promise != null) await promise;
  } catch (exc) {
    const stat = parseException(exc, message, skip);
    if (stat != null) return add(stat);
  }
};

const createErrorHandler =
  (add: Adder): ErrorHandler =>
  (excOrFunc, message, skip): void => {
    if (!handleException(excOrFunc, add, message, skip)) return;
    void handleFunc(excOrFunc, add, message, skip);
  };

const createAsyncErrorHandler =
  (add: Adder): AsyncErrorHandler =>
  async (func, message, skip): Promise<void> => {
    if (!handleException(func, add, message, skip)) return;
    await handleFunc(func, add, message, skip);
  };

const newWrapper = (client: Client | null, fluxClient: Flux.Client) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Status.Aggregator>
      <Synnax.TestProvider client={client}>
        <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
      </Synnax.TestProvider>
    </Status.Aggregator>
  );
  return Wrapper;
};

export interface CreateSynnaxWrapperArgs {
  client: Client | null;
  excludeFluxStores?: string[];
}

const createFluxClient = (args: CreateSynnaxWrapperArgs): Flux.Client => {
  const { client, excludeFluxStores } = args;
  const storeConfig = { ...Pluto.FLUX_STORE_CONFIG };
  if (excludeFluxStores)
    excludeFluxStores.forEach((store) => delete storeConfig[store]);
  return new Flux.Client({
    client,
    storeConfig,
    handleError: createErrorHandler(console.error),
    handleAsyncError: createAsyncErrorHandler(console.error),
  });
};

export const createSynnaxWrapper = ({
  client,
  excludeFluxStores,
}: CreateSynnaxWrapperArgs): FC<PropsWithChildren> =>
  newWrapper(client, createFluxClient({ client, excludeFluxStores }));

export const createAsyncSynnaxWrapper = async (
  args: CreateSynnaxWrapperArgs,
): Promise<FC<PropsWithChildren>> => {
  const fluxClient = createFluxClient(args);
  await fluxClient.awaitInitialized();
  return newWrapper(args.client, fluxClient);
};
