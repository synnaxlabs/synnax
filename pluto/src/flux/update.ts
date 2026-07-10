// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status, type Synnax as Client } from "@synnaxlabs/client";
import { type CrudeTimeSpan, type destructor } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import type z from "zod";

import { type base } from "@/flux/base";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  type InitialStatusDetailsContainer,
  loadingResult,
  nullClientResult,
  parseInitialStatusDetails,
  type Result,
  type ResultStatus,
  resultStatusDetails,
  successResult,
} from "@/flux/result";
import { useDebouncedCallback } from "@/hooks";
import { state } from "@/state";
import { useAdder } from "@/status/base/Aggregator";
import { Synnax } from "@/synnax";

export interface UpdateParams<
  Input extends base.Data,
  Store extends base.Store,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> {
  data: Input;
  client: AllowDisconnected extends true ? Client | null : Client;
  store: Store;
  rollbacks: destructor.Destructor[];
  setStatus: (setter: state.SetArg<ResultStatus<StatusDetails>>) => void;
  onOptimisticComplete: (data: Output) => Promise<void>;
}

export type CreateUpdateParams<
  Input extends base.Data,
  ScopedStore extends base.Store,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> = {
  name: string;
  verbs: base.Verbs;
  update: (
    params: UpdateParams<Input, ScopedStore, Output, StatusDetails, AllowDisconnected>,
  ) => Promise<Output | false>;
  allowDisconnected?: AllowDisconnected;
} & InitialStatusDetailsContainer<StatusDetails>;

/**
 * UpdateOptions is the second argument to the update methods returned by a flux
 * update hook. extra is carried on the options object alongside the standard
 * fetch options and is threaded into beforeUpdate, afterOptimistic,
 * afterSuccess, and afterFailure so extension callers can pass per-call context
 * into their own lifecycle callbacks.
 */
export type UpdateOptions<Extra = void> = base.FetchOptions & { extra?: Extra };

export interface UseObservableUpdateReturn<Input extends base.Data, Extra = void> {
  update: (data: Input, opts?: UpdateOptions<Extra>) => void;
  updateAsync: (data: Input, opts?: UpdateOptions<Extra>) => Promise<boolean>;
}

export interface UseObservableUpdateParams<
  Input extends base.Data,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  SubStore extends base.Store = {},
  Extra = void,
> {
  debounce?: CrudeTimeSpan;
  onChange: state.Setter<Result<Input | undefined, StatusDetails>>;
  scope?: string;
  beforeUpdate?: (
    params: BeforeUpdateParams<Input, AllowDisconnected, SubStore, Extra>,
  ) => Promise<Input | boolean> | Input | boolean;
  afterOptimistic?: (
    params: AfterOptimisticParams<Output, AllowDisconnected, SubStore, Extra>,
  ) => Promise<void> | void;
  afterSuccess?: (
    params: AfterSuccessParams<Output, AllowDisconnected, Extra>,
  ) => Promise<void> | void;
  afterFailure?: (
    params: AfterFailureParams<Input, AllowDisconnected, Extra>,
  ) => Promise<void> | void;
}

export interface BeforeUpdateParams<
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
  Store extends base.Store = {},
  Extra = void,
> {
  rollbacks: destructor.Destructor[];
  client: AllowDisconnected extends true ? Client | null : Client;
  data: Data;
  store: Store;
  extra: Extra;
}

export interface AfterOptimisticParams<
  Output extends base.Data,
  AllowDisconnected extends boolean = false,
  Store extends base.Store = {},
  Extra = void,
> {
  rollbacks: destructor.Destructor[];
  client: AllowDisconnected extends true ? Client | null : Client;
  data: Output;
  store: Store;
  extra: Extra;
}

export interface AfterSuccessParams<
  Output extends base.Data,
  AllowDisconnected extends boolean = false,
  Extra = void,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  data: Output;
  extra: Extra;
}

export interface AfterFailureParams<
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
  Extra = void,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  status: status.Status<typeof status.exceptionDetailsSchema, z.ZodLiteral<"error">>;
  data: Data;
  extra: Extra;
}

export interface UseDirectUpdateParams<
  Input extends base.Data,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  SubStore extends base.Store = {},
  Extra = void,
> extends Omit<
  UseObservableUpdateParams<
    Input,
    Output,
    StatusDetails,
    AllowDisconnected,
    SubStore,
    Extra
  >,
  "onChange"
> {}

export type UseDirectUpdateReturn<
  Input extends base.Data,
  StatusDetails extends z.ZodType = z.ZodNever,
  Extra = void,
> = Result<Input | undefined, StatusDetails> & UseObservableUpdateReturn<Input, Extra>;

export interface UseObservableUpdate<
  Input extends base.Data,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  SubStore extends base.Store = {},
> {
  (
    args: UseObservableUpdateParams<
      Input,
      Output,
      StatusDetails,
      AllowDisconnected,
      SubStore
    >,
  ): UseObservableUpdateReturn<Input>;
  <Extra>(
    args: UseObservableUpdateParams<
      Input,
      Output,
      StatusDetails,
      AllowDisconnected,
      SubStore,
      Extra
    >,
  ): UseObservableUpdateReturn<Input, Extra>;
}

export interface UseUpdate<
  Input extends base.Data,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  SubStore extends base.Store = {},
> {
  (
    args?: UseDirectUpdateParams<
      Input,
      Output,
      StatusDetails,
      AllowDisconnected,
      SubStore
    >,
  ): UseDirectUpdateReturn<Input, StatusDetails>;
  <Extra>(
    args?: UseDirectUpdateParams<
      Input,
      Output,
      StatusDetails,
      AllowDisconnected,
      SubStore,
      Extra
    >,
  ): UseDirectUpdateReturn<Input, StatusDetails, Extra>;
}

export interface CreateUpdateReturn<
  Input extends base.Data,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  SubStore extends base.Store = {},
> {
  useObservableUpdate: UseObservableUpdate<
    Input,
    Output,
    StatusDetails,
    AllowDisconnected,
    SubStore
  >;
  useUpdate: UseUpdate<Input, Output, StatusDetails, AllowDisconnected, SubStore>;
}

const useObservable = <
  Input extends base.Data,
  Store extends base.Store,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  Extra = void,
>(
  params: UseObservableUpdateParams<
    Input,
    Output,
    StatusDetails,
    AllowDisconnected,
    Store,
    Extra
  > &
    CreateUpdateParams<Input, Store, Output, StatusDetails, AllowDisconnected>,
): UseObservableUpdateReturn<Input, Extra> => {
  const {
    onChange,
    update,
    name,
    verbs: { present, past, participle },
    debounce = 0,
    scope,
    beforeUpdate,
    afterOptimistic,
    afterSuccess,
    afterFailure,
    allowDisconnected = false as AllowDisconnected,
  } = params;
  const maybeClient = Synnax.use();
  const store = useStore<Store>(scope);
  const addStatus = useAdder();
  const runUpdate = useCallback(
    async (
      data: Input,
      opts: base.FetchOptions & { extra?: Extra } = {},
    ): Promise<boolean> => {
      const { signal, extra: rawExtra } = opts;
      const extra: Extra = rawExtra as Extra;

      const rollbacks: destructor.Destructor[] = [];
      const runRollbacks = () => {
        try {
          rollbacks.reverse().forEach((rollback) => rollback());
        } catch (error) {
          console.error(`failed to rollback changes to ${name}`, error);
        }
      };

      if (maybeClient == null && !allowDisconnected) {
        onChange((p) =>
          nullClientResult(
            `${present} ${name}`,
            resultStatusDetails<Input | undefined, StatusDetails>(p),
          ),
        );
        return false;
      }

      const client = maybeClient as AllowDisconnected extends true
        ? Client | null
        : Client;

      try {
        onChange((p) =>
          loadingResult(
            `${participle} ${name}`,
            p.data,
            resultStatusDetails<Input | undefined, StatusDetails>(p),
          ),
        );

        if (beforeUpdate != null) {
          const updatedValue = await beforeUpdate({
            client,
            data,
            rollbacks,
            store,
            extra,
          });
          if (signal?.aborted === true) return false;
          if (updatedValue === false) {
            onChange(successResult(`${past} ${name}`, data));
            runRollbacks();
            return false;
          }
          if (updatedValue !== true) data = updatedValue;
        }

        const setStatus = (setter: state.SetArg<ResultStatus<StatusDetails>>) =>
          onChange((p) => {
            const nextStatus = state.executeSetter(setter, p.status);
            return {
              ...p,
              status: nextStatus,
              variant: nextStatus.variant,
            } as Result<Input | undefined, StatusDetails>;
          });

        const onOptimisticComplete = async (output: Output): Promise<void> => {
          if (signal?.aborted === true) return;
          await afterOptimistic?.({ client, data: output, store, rollbacks, extra });
        };

        const output = await update({
          client,
          data,
          store,
          rollbacks,
          setStatus,
          onOptimisticComplete,
        });
        if (signal?.aborted === true) return false;
        onChange((p) =>
          successResult(
            `${past} ${name}`,
            data,
            resultStatusDetails<Input | undefined, StatusDetails>(p),
          ),
        );
        if (output === false) return false;
        await afterSuccess?.({ client, data: output, extra });
        return true;
      } catch (error) {
        runRollbacks();
        if (signal?.aborted === true) return false;

        const result = errorResult(`${present} ${name}`, error);
        const { status } = result;
        onChange(result);
        addStatus(status);
        await afterFailure?.({ client, status, data, extra });
        return false;
      }
    },
    [
      maybeClient,
      allowDisconnected,
      name,
      present,
      participle,
      past,
      store,
      onChange,
      addStatus,
      update,
      beforeUpdate,
      afterOptimistic,
      afterSuccess,
      afterFailure,
    ],
  );
  const handleUpdate = useDebouncedCallback(
    (data: Input, opts?: base.FetchOptions & { extra?: Extra }) => {
      void runUpdate(data, opts);
    },
    debounce,
    [runUpdate],
  );
  return {
    update: handleUpdate,
    updateAsync: runUpdate,
  };
};

const useDirect = <
  Input extends base.Data,
  Store extends base.Store = {},
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
  Extra = void,
>(
  params: UseDirectUpdateParams<
    Input,
    Output,
    StatusDetails,
    AllowDisconnected,
    Store,
    Extra
  > &
    CreateUpdateParams<Input, Store, Output, StatusDetails, AllowDisconnected>,
): UseDirectUpdateReturn<Input, StatusDetails, Extra> => {
  const { name, verbs, ...restParams } = params;
  const initialStatusDetails = parseInitialStatusDetails<StatusDetails>(params);
  const [result, setResult] = useState<Result<Input | undefined, StatusDetails>>(
    successResult<Input | undefined, StatusDetails>(
      `${verbs.past} ${name}`,
      undefined,
      initialStatusDetails,
    ),
  );
  const methods = useObservable<
    Input,
    Store,
    Output,
    StatusDetails,
    AllowDisconnected,
    Extra
  >({
    ...restParams,
    initialStatusDetails,
    verbs,
    name,
    onChange: setResult,
  });
  return { ...result, ...methods };
};

export const createUpdate = <
  Input extends base.Data,
  ScopedStore extends base.Store,
  Output extends base.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
>(
  createParams: CreateUpdateParams<
    Input,
    ScopedStore,
    Output,
    StatusDetails,
    AllowDisconnected
  >,
): CreateUpdateReturn<
  Input,
  Output,
  StatusDetails,
  AllowDisconnected,
  ScopedStore
> => ({
  useObservableUpdate: <Extra = void>(
    params: UseObservableUpdateParams<
      Input,
      Output,
      StatusDetails,
      AllowDisconnected,
      ScopedStore,
      Extra
    >,
  ) =>
    useObservable<Input, ScopedStore, Output, StatusDetails, AllowDisconnected, Extra>({
      ...params,
      ...createParams,
    }),
  useUpdate: <Extra = void>(
    params: UseDirectUpdateParams<
      Input,
      Output,
      StatusDetails,
      AllowDisconnected,
      ScopedStore,
      Extra
    > = {},
  ) =>
    useDirect<Input, ScopedStore, Output, StatusDetails, AllowDisconnected, Extra>({
      ...params,
      ...createParams,
    }),
});
