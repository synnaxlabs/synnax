// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, type status, type Synnax as Client } from "@synnaxlabs/client";
import { type CrudeTimeSpan, type destructor, state, type verbs } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import type z from "zod";

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
import { useAdder } from "@/status/base/Aggregator";
import { Synnax } from "@/synnax";

export interface UpdateParams<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  data: Input;
  client: Client;
  setStatus: (setter: state.SetArg<ResultStatus<StatusDetails>>) => void;
  onOptimisticComplete: (data: Output) => Promise<void>;
}

export type CreateUpdateParams<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> = {
  name: string;
  verbs: verbs.Verbs;
  update: (
    params: UpdateParams<Input, Output, StatusDetails>,
  ) => Promise<Output | false>;
} & InitialStatusDetailsContainer<StatusDetails>;

export interface UseObservableUpdateReturn<Input extends query.Data> {
  update: (data: Input, opts?: query.FetchOptions) => void;
  updateAsync: (data: Input, opts?: query.FetchOptions) => Promise<boolean>;
}

export interface UseObservableUpdateParams<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  debounce?: CrudeTimeSpan;
  onChange: state.Setter<Result<Input | undefined, StatusDetails>>;
  beforeUpdate?: (
    params: BeforeUpdateParams<Input>,
  ) => Promise<Input | boolean> | Input | boolean;
  afterOptimistic?: (params: AfterOptimisticParams<Output>) => Promise<void> | void;
  afterSuccess?: (params: AfterSuccessParams<Output>) => Promise<void> | void;
  afterFailure?: (params: AfterFailureParams<Input>) => Promise<void> | void;
}

export interface BeforeUpdateParams<Data extends query.Data> {
  /** Side-effect undos run in reverse order when the update fails. */
  rollbacks: destructor.Destructor[];
  client: Client;
  data: Data;
}

export interface AfterOptimisticParams<Output extends query.Data> {
  /** Side-effect undos run in reverse order when the update fails. */
  rollbacks: destructor.Destructor[];
  client: Client;
  data: Output;
}

export interface AfterSuccessParams<Output extends query.Data> {
  client: Client;
  data: Output;
}

export interface AfterFailureParams<Data extends query.Data> {
  client: Client;
  status: status.Status<typeof status.exceptionDetailsSchema, z.ZodLiteral<"error">>;
  data: Data;
}

export interface UseDirectUpdateParams<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> extends Omit<UseObservableUpdateParams<Input, Output, StatusDetails>, "onChange"> {}

export type UseDirectUpdateReturn<
  Input extends query.Data,
  StatusDetails extends z.ZodType = z.ZodNever,
> = Result<Input | undefined, StatusDetails> & UseObservableUpdateReturn<Input>;

export interface UseObservableUpdate<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  (
    params: UseObservableUpdateParams<Input, Output, StatusDetails>,
  ): UseObservableUpdateReturn<Input>;
}

export interface UseUpdate<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  (
    params?: UseDirectUpdateParams<Input, Output, StatusDetails>,
  ): UseDirectUpdateReturn<Input, StatusDetails>;
}

export interface CreateUpdateReturn<
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
> {
  useObservableUpdate: UseObservableUpdate<Input, Output, StatusDetails>;
  useUpdate: UseUpdate<Input, Output, StatusDetails>;
}

const useObservable = <
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  params: UseObservableUpdateParams<Input, Output, StatusDetails> &
    CreateUpdateParams<Input, Output, StatusDetails>,
): UseObservableUpdateReturn<Input> => {
  const {
    onChange,
    update,
    name,
    verbs: { present, past, participle },
    debounce = 0,
    beforeUpdate,
    afterOptimistic,
    afterSuccess,
    afterFailure,
  } = params;
  const client = Synnax.use();
  const addStatus = useAdder();
  const runUpdate = useCallback(
    async (data: Input, opts: query.FetchOptions = {}): Promise<boolean> => {
      const { signal } = opts;

      const rollbacks: destructor.Destructor[] = [];
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

        const onOptimisticComplete = async (output: Output): Promise<void> => {
          if (signal?.aborted === true) return;
          await afterOptimistic?.({ client, data: output, rollbacks });
        };

        const output = await update({ client, data, setStatus, onOptimisticComplete });
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
    [
      client,
      name,
      present,
      participle,
      past,
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
    (data: Input, opts?: query.FetchOptions) => {
      void runUpdate(data, opts);
    },
    debounce,
    [runUpdate],
  );
  return { update: handleUpdate, updateAsync: runUpdate };
};

const useDirect = <
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  params: UseDirectUpdateParams<Input, Output, StatusDetails> &
    CreateUpdateParams<Input, Output, StatusDetails>,
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
  const methods = useObservable<Input, Output, StatusDetails>({
    ...restParams,
    initialStatusDetails,
    verbs,
    name,
    onChange: setResult,
  });
  return { ...result, ...methods };
};

export const createUpdate = <
  Input extends query.Data,
  Output extends query.Data = Input,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  createParams: CreateUpdateParams<Input, Output, StatusDetails>,
): CreateUpdateReturn<Input, Output, StatusDetails> => ({
  useObservableUpdate: (
    params: UseObservableUpdateParams<Input, Output, StatusDetails>,
  ) =>
    useObservable<Input, Output, StatusDetails>({
      ...params,
      ...createParams,
    }),
  useUpdate: (params: UseDirectUpdateParams<Input, Output, StatusDetails> = {}) =>
    useDirect<Input, Output, StatusDetails>({
      ...params,
      ...createParams,
    }),
});
