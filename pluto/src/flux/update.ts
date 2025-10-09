// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type Destructor, type status } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import type z from "zod";

import { type core } from "@/flux/core";
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
import { useAdder } from "@/status/core/Aggregator";
import { Synnax } from "@/synnax";

export interface UpdateParams<
  Input extends core.Shape,
  Store extends core.Store,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  data: Input;
  client: Client;
  store: Store;
  rollbacks: Destructor[];
  setStatus: (setter: state.SetArg<ResultStatus<StatusDetails>>) => void;
}

export type CreateUpdateParams<
  Input extends core.Shape,
  ScopedStore extends core.Store,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> = {
  name: string;
  verbs: core.Verbs;
  update: (
    params: UpdateParams<Input, ScopedStore, StatusDetails>,
  ) => Promise<Output | false>;
} & InitialStatusDetailsContainer<StatusDetails>;

export interface UseObservableUpdateReturn<Input extends core.Shape> {
  update: (data: Input, opts?: core.FetchOptions) => void;
  updateAsync: (data: Input, opts?: core.FetchOptions) => Promise<boolean>;
}

export interface UseObservableUpdateParams<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  debounce?: number;
  onChange: state.Setter<Result<Input | undefined, StatusDetails>>;
  scope?: string;
  beforeUpdate?: (
    params: BeforeUpdateParams<Input>,
  ) => Promise<Input | boolean> | Input | boolean;
  afterSuccess?: (params: AfterSuccessParams<Output>) => Promise<void> | void;
  afterFailure?: (params: AfterFailureParams<Input>) => Promise<void> | void;
}

export interface BeforeUpdateParams<Data extends core.Shape> {
  rollbacks: Destructor[];
  client: Client;
  data: Data;
}

export interface AfterSuccessParams<Output extends core.Shape> {
  client: Client;
  data: Output;
}

export interface AfterFailureParams<Data extends core.Shape> {
  client: Client;
  status: status.Status<typeof status.exceptionDetailsSchema, "error">;
  data: Data;
}

export interface UseDirectUpdateParams<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> extends Omit<UseObservableUpdateParams<Input, Output, StatusDetails>, "onChange"> {}

export type UseDirectUpdateReturn<
  Input extends core.Shape,
  StatusDetails extends z.ZodType = z.ZodNever,
> = Result<Input | undefined, StatusDetails> & UseObservableUpdateReturn<Input>;

export interface UseObservableUpdate<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  (
    args: UseObservableUpdateParams<Input, Output, StatusDetails>,
  ): UseObservableUpdateReturn<Input>;
}

export interface UseUpdate<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  (
    args?: UseDirectUpdateParams<Input, Output, StatusDetails>,
  ): UseDirectUpdateReturn<Input, StatusDetails>;
}

export interface CreateUpdateReturn<
  Input extends core.Shape,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  useObservableUpdate: UseObservableUpdate<Input, Output, StatusDetails>;
  useUpdate: UseUpdate<Input, Output, StatusDetails>;
}

const useObservable = <
  Input extends core.Shape,
  Store extends core.Store,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  params: UseObservableUpdateParams<Input, Output, StatusDetails> &
    CreateUpdateParams<Input, Store, Output, StatusDetails>,
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
  } = params;
  const client = Synnax.use();
  const store = useStore<Store>(scope);
  const addStatus = useAdder();
  const handleUpdate = useDebouncedCallback(
    async (data: Input, opts: core.FetchOptions = {}): Promise<boolean> => {
      const { signal } = opts;

      const rollbacks: Destructor[] = [];
      const runRollbacks = () => {
        try {
          rollbacks.reverse().forEach((rollback) => rollback());
        } catch (error) {
          console.error(`failed to rollback changes to ${name}`, error);
        }
      };

      if (client == null) {
        onChange((p) =>
          nullClientResult(
            `${present} ${name}`,
            resultStatusDetails<Input | undefined, StatusDetails>(p),
          ),
        );
        return false;
      }

      try {
        onChange((p) =>
          loadingResult(
            `${participle} ${name}`,
            p.data,
            resultStatusDetails<Input | undefined, StatusDetails>(p),
          ),
        );

        if (beforeUpdate != null) {
          const updatedValue = await beforeUpdate({ client, data, rollbacks });
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
    (data: Input, opts?: core.FetchOptions) => void handleUpdate(data, opts),
    [handleUpdate],
  );
  return { update: handleSyncUpdate, updateAsync: handleUpdate };
};

const useDirect = <
  Input extends core.Shape,
  Store extends core.Store = {},
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  params: UseDirectUpdateParams<Input, Output, StatusDetails> &
    CreateUpdateParams<Input, Store, Output, StatusDetails>,
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
  const methods = useObservable<Input, Store, Output, StatusDetails>({
    ...restParams,
    initialStatusDetails,
    verbs,
    name,
    onChange: setResult,
  });
  return { ...result, ...methods };
};

export const createUpdate = <
  Input extends core.Shape,
  ScopedStore extends core.Store,
  Output extends core.Shape = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  createParams: CreateUpdateParams<Input, ScopedStore, Output, StatusDetails>,
): CreateUpdateReturn<Input, Output, StatusDetails> => ({
  useObservableUpdate: (
    params: UseObservableUpdateParams<Input, Output, StatusDetails>,
  ) =>
    useObservable<Input, ScopedStore, Output, StatusDetails>({
      ...params,
      ...createParams,
    }),
  useUpdate: (params: UseDirectUpdateParams<Input, Output, StatusDetails> = {}) =>
    useDirect<Input, ScopedStore, Output, StatusDetails>({
      ...params,
      ...createParams,
    }),
});
