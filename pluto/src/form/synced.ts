// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax as Client } from "@synnaxlabs/client";
import { observe, Primitive } from "@synnaxlabs/x";
import { useCallback } from "react";
import { z } from "zod";

import { ContextValue, OnChangeProps, use, UseProps, UseReturn } from "@/form/Form";
import { useAsyncEffect } from "@/hooks";
import { useMemoPrimitiveArray } from "@/memo";
import { Observe } from "@/observe";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

interface QueryFnProps {
  client: Client;
}

interface ApplyObservableProps<Z extends z.ZodTypeAny, O = Z> {
  changes: O;
  ctx: ContextValue<Z>;
}

interface SyncLocalProps<Z extends z.ZodTypeAny> extends OnChangeProps<Z> {
  client: Client;
}

interface UseSyncedProps<Z extends z.ZodTypeAny, O = Z> extends UseProps<Z> {
  name: string;
  key: Primitive[];
  queryFn: (props: QueryFnProps) => Promise<z.output<Z>>;
  openObservable?: (client: Client) => Promise<observe.ObservableAsyncCloseable<O>>;
  applyObservable?: (props: ApplyObservableProps<Z, O>) => void;
  applyChanges?: (props: SyncLocalProps<Z>) => Promise<void>;
}

export const useSynced = <Z extends z.ZodTypeAny, O = Z>({
  key,
  name,
  queryFn,
  values: initialValues,
  openObservable,
  applyChanges,
  applyObservable,
  ...props
}: UseSyncedProps<Z, O>): UseReturn<Z> => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const memoKey = useMemoPrimitiveArray(key);

  const methods = use({
    values: initialValues,
    ...props,
    sync: false,
    onChange: (props) => {
      if (client == null) return;
      void (async () => {
        try {
          await applyChanges?.({ ...props, client });
        } catch (error) {
          addStatus({
            variant: "error",
            message: `Failed to save ${name}`,
            description: (error as Error).message,
          });
        }
      })();
    },
  });
  useAsyncEffect(async () => {
    if (client == null) return;
    try {
      const values = await queryFn({ client });
      methods.set("", values);
    } catch (error) {
      addStatus({
        variant: "error",
        message: `Failed to retrieve ${name}`,
        description: (error as Error).message,
      });
    }
  }, [memoKey, client?.key]);
  const onOpenObs = useCallback(async () => {
    if (client == null) return;
    return await openObservable?.(client);
  }, [memoKey, client?.key]);
  const onObsChange = useCallback(
    (value: O) => applyObservable?.({ changes: value, ctx: methods }),
    [memoKey, applyObservable],
  );
  Observe.useListener({ key: memoKey, open: onOpenObs, onChange: onObsChange });
  return methods;
};