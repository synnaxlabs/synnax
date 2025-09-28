// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type observe, type primitive } from "@synnaxlabs/x";
import { useCallback } from "react";
import { type z } from "zod";

import { type ContextValue } from "@/form/Context";
import { type OnChangeArgs, use, type UseArgs, type UseReturn } from "@/form/use";
import { useAsyncEffect } from "@/hooks";
import { useMemoPrimitiveArray } from "@/memo";
import { Observe } from "@/observe";
import { Status } from "@/status/core";
import { Synnax } from "@/synnax";

interface QueryFnProps {
  client: Client;
}

interface ApplyObservableProps<Z extends z.ZodType, O = Z> {
  changes: O;
  ctx: ContextValue<Z>;
}

interface SyncLocalProps<Z extends z.ZodType> extends OnChangeArgs<Z> {
  client: Client;
}

interface UseSyncedProps<Z extends z.ZodType, O = Z> extends UseArgs<Z> {
  name: string;
  key: primitive.Value[];
  queryFn: (props: QueryFnProps) => Promise<z.infer<Z>>;
  openObservable?: (client: Client) => Promise<observe.ObservableAsyncCloseable<O>>;
  applyObservable?: (props: ApplyObservableProps<Z, O>) => void;
  applyChanges?: (props: SyncLocalProps<Z>) => Promise<void>;
}

export const useSynced = <Z extends z.ZodType, O = Z>({
  key,
  name,
  queryFn,
  values: initialValues,
  openObservable,
  applyChanges,
  applyObservable,
  ...rest
}: UseSyncedProps<Z, O>): UseReturn<Z> => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const memoKey = useMemoPrimitiveArray(key);

  const methods = use({
    values: initialValues,
    ...rest,
    sync: false,
    onChange: (props) => {
      if (client == null) return;
      handleError(async () => {
        await applyChanges?.({ ...props, client });
      }, `Failed to apply changes for ${name}`);
    },
  });
  useAsyncEffect(async () => {
    if (client == null) return;
    try {
      const values = await queryFn({ client });
      methods.set("", values);
    } catch (e) {
      handleError(e, `Failed to retrieve ${name}`);
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
