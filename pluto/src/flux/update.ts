// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type destructor, type status } from "@synnaxlabs/x";
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
  Input extends base.Shape,
  Store extends base.Store,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> {
  data: Input;
  client: AllowDisconnected extends true ? Client | null : Client;
  store: Store;
  rollbacks: destructor.Destructor[];
  setStatus: (setter: state.SetArg<ResultStatus<StatusDetails>>) => void;
}

export type CreateUpdateParams<
  Input extends base.Shape,
  ScopedStore extends base.Store,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> = {
  name: string;
  verbs: base.Verbs;
  update: (
    params: UpdateParams<Input, ScopedStore, StatusDetails, AllowDisconnected>,
  ) => Promise<Output | false>;
  allowDisconnected?: AllowDisconnected;
} & InitialStatusDetailsContainer<StatusDetails>;

export interface UseObservableUpdateReturn<Input extends base.Shape> {
  update: (data: Input, opts?: base.FetchOptions) => void;
  updateAsync: (data: Input, opts?: base.FetchOptions) => Promise<boolean>;
}

export interface UseObservableUpdateParams<
  Input extends base.Shape,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> {
  debounce?: number;
  onChange: state.Setter<Result<Input | undefined, StatusDetails>>;
  scope?: string;
  beforeUpdate?: (
    params: BeforeUpdateParams<Input, AllowDisconnected>,
  ) => Promise<Input | boolean> | Input | boolean;
  afterSuccess?: (
    params: AfterSuccessParams<Output, AllowDisconnected>,
  ) => Promise<void> | void;
  afterFailure?: (
    params: AfterFailureParams<Input, AllowDisconnected>,
  ) => Promise<void> | void;
}

export interface BeforeUpdateParams<
  Data extends base.Shape,
  AllowDisconnected extends boolean = false,
> {
  rollbacks: destructor.Destructor[];
  client: AllowDisconnected extends true ? Client | null : Client;
  data: Data;
}

export interface AfterSuccessParams<
  Output extends base.Shape,
  AllowDisconnected extends boolean = false,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  data: Output;
}

export interface AfterFailureParams<
  Data extends base.Shape,
  AllowDisconnected extends boolean = false,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  status: status.Status<typeof status.exceptionDetailsSchema, "error">;
  data: Data;
}

export interface UseDirectUpdateParams<
  Input extends base.Shape,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> extends Omit<
  UseObservableUpdateParams<Input, Output, StatusDetails, AllowDisconnected>,
  "onChange"
> {}

export type UseDirectUpdateReturn<
  Input extends base.Shape,
  StatusDetails extends z.ZodType = z.ZodNever,
> = Result<Input | undefined, StatusDetails> & UseObservableUpdateReturn<Input>;

export interface UseObservableUpdate<
  Input extends base.Shape,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> {
  (
    args: UseObservableUpdateParams<Input, Output, StatusDetails, AllowDisconnected>,
  ): UseObservableUpdateReturn<Input>;
}

export interface UseUpdate<
  Input extends base.Shape,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> {
  (
    args?: UseDirectUpdateParams<Input, Output, StatusDetails, AllowDisconnected>,
  ): UseDirectUpdateReturn<Input, StatusDetails>;
}

export interface CreateUpdateReturn<
  Input extends base.Shape,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
> {
  useObservableUpdate: UseObservableUpdate<
    Input,
    Output,
    StatusDetails,
    AllowDisconnected
  >;
  useUpdate: UseUpdate<Input, Output, StatusDetails, AllowDisconnected>;
}

const useObservable = <
  Input extends base.Shape,
  Store extends base.Store,
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
>(
  params: UseObservableUpdateParams<Input, Output, StatusDetails, AllowDisconnected> &
    CreateUpdateParams<Input, Store, Output, StatusDetails, AllowDisconnected>,
): UseObservableUpdateReturn<Input> => {
  const {
    onChange,
    update,
    name,
    verbs: { present, past, participle },
    debounce = 0,
    scope,
    beforeUpdate,
    afterSuccess,
    afterFailure,
    allowDisconnected = false as AllowDisconnected,
  } = params;
  const maybeClient = Synnax.use();
  const store = useStore<Store>(scope);
  const addStatus = useAdder();
  const handleUpdate = useDebouncedCallback(
    async (data: Input, opts: base.FetchOptions = {}): Promise<boolean> => {
      const { signal } = opts;

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

        const output = await update({ client, data, store, rollbacks, setStatus });
        if (signal?.aborted === true) return false;
        onChange((p) =>
          successResult(
            `${past} ${name}`,
            data,
            resultStatusDetails<Input | undefined, StatusDetails>(p),
          ),
        );
        if (output === false) return false;
        await afterSuccess?.({ client, data: output });
        return true;
      } catch (error) {
        runRollbacks();
        if (signal?.aborted === true) return false;

        const result = errorResult(`${present} ${name}`, error);
        const { status } = result;
        onChange(result);
        addStatus(status);
        await afterFailure?.({ client, status, data });
        return false;
      }
    },
    debounce,
    [name, onChange, beforeUpdate, afterSuccess, afterFailure],
  );
  const handleSyncUpdate = useCallback(
    (data: Input, opts?: base.FetchOptions) => void handleUpdate(data, opts),
    [handleUpdate],
  );
  return { update: handleSyncUpdate, updateAsync: handleUpdate };
};

const useDirect = <
  Input extends base.Shape,
  Store extends base.Store = {},
  Output extends base.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
  AllowDisconnected extends boolean = false,
>(
  params: UseDirectUpdateParams<Input, Output, StatusDetails, AllowDisconnected> &
    CreateUpdateParams<Input, Store, Output, StatusDetails, AllowDisconnected>,
): UseDirectUpdateReturn<Input, StatusDetails> => {
  const { name, verbs, ...restParams } = params;
  const initialStatusDetails = parseInitialStatusDetails<StatusDetails>(params);
  const [result, setResult] = useState<Result<Input | undefined, StatusDetails>>(
    successResult<Input | undefined, StatusDetails>(
      `${verbs.past} ${name}`,
      undefined,
      initialStatusDetails,
    ),
  );
  const methods = useObservable<Input, Store, Output, StatusDetails, AllowDisconnected>(
    {
      ...restParams,
      initialStatusDetails,
      verbs,
      name,
      onChange: setResult,
    },
  );
  return { ...result, ...methods };
};

export const createUpdate = <
  Input extends base.Shape,
  ScopedStore extends base.Store,
  Output extends base.Shape = Input,
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
): CreateUpdateReturn<Input, Output, StatusDetails, AllowDisconnected> => ({
  useObservableUpdate: (
    params: UseObservableUpdateParams<Input, Output, StatusDetails, AllowDisconnected>,
  ) =>
    useObservable<Input, ScopedStore, Output, StatusDetails, AllowDisconnected>({
      ...params,
      ...createParams,
    }),
  useUpdate: (
    params: UseDirectUpdateParams<Input, Output, StatusDetails, AllowDisconnected> = {},
  ) =>
    useDirect<Input, ScopedStore, Output, StatusDetails, AllowDisconnected>({
      ...params,
      ...createParams,
    }),
});
