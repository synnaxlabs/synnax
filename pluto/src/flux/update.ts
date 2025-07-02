import { type Synnax } from "@synnaxlabs/client";
import { useCallback, useState } from "react";

import { type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

export interface UpdateArgs<UpdateParams extends Params, Data extends state.State> {
  value: Data;
  params: UpdateParams;
  /** The Synnax client instance for making requests */
  client: Synnax;
  /** Function to update the form state with new data */
  onChange: state.PureSetter<Data>;
}

export interface CreateUpdateArgs<
  UpdateParams extends Params,
  Data extends state.State,
> {
  name: string;
  update: (args: UpdateArgs<UpdateParams, Data>) => Promise<void>;
}

export interface UseObservableUpdateReturn<Data extends state.State> {
  update: (value: Data) => void;
  updateAsync: (value: Data) => Promise<void>;
}

export interface UseObservableUpdateArgs<
  UpdateParams extends Params,
  Data extends state.State,
> {
  onChange: state.Setter<Result<Data | null>>;
  params: UpdateParams;
}

export interface UseDirectUpdateArgs<UpdateParams extends Params> {
  params: UpdateParams;
}

export type UseDirectUpdateReturn<Data extends state.State> = Result<Data | null> &
  UseObservableUpdateReturn<Data>;

export interface CreateUpdateReturn<
  UpdateParams extends Params,
  Data extends state.State,
> {
  useObservable: (
    args: UseObservableUpdateArgs<UpdateParams, Data>,
  ) => UseObservableUpdateReturn<Data>;
  useDirect: (args: UseDirectUpdateArgs<UpdateParams>) => UseDirectUpdateReturn<Data>;
}

const useObservable = <UpdateParams extends Params, Data extends state.State>({
  onChange,
  params,
  update,
  name,
}: UseObservableUpdateArgs<UpdateParams, Data> &
  CreateUpdateArgs<UpdateParams, Data>): UseObservableUpdateReturn<Data> => {
  const client = PSynnax.use();
  const handleUpdate = useCallback(
    async (value: Data) => {
      try {
        if (client == null) return onChange(nullClientResult(name, "update"));
        onChange(pendingResult(name, "updating"));
        await update({
          client,
          onChange: (value) => onChange(successResult(name, "updated", value)),
          value,
          params,
        });
        onChange(successResult(name, "updated", value));
      } catch (error) {
        onChange(errorResult(name, "update", error));
      }
    },
    [name, params],
  );
  const handleSyncUpdate = useCallback(
    (value: Data) => void handleUpdate(value),
    [handleUpdate],
  );
  return {
    update: handleSyncUpdate,
    updateAsync: handleUpdate,
  };
};

const useDirect = <UpdateParams extends Params, Data extends state.State>({
  params,
  name,
  ...restArgs
}: UseDirectUpdateArgs<UpdateParams> &
  CreateUpdateArgs<UpdateParams, Data>): UseDirectUpdateReturn<Data> => {
  const [result, setResult] = useState<Result<Data | null>>(
    successResult(name, "updated", null),
  );
  const ret = useObservable<UpdateParams, Data>({
    ...restArgs,
    name,
    onChange: setResult,
    params,
  });
  return { ...result, ...ret };
};

export const createUpdate = <UpdateParams extends Params, Data extends state.State>(
  createArgs: CreateUpdateArgs<UpdateParams, Data>,
): CreateUpdateReturn<UpdateParams, Data> => ({
  useObservable: (args: UseObservableUpdateArgs<UpdateParams, Data>) =>
    useObservable({ ...args, ...createArgs }),
  useDirect: (args: UseDirectUpdateArgs<UpdateParams>) =>
    useDirect({ ...args, ...createArgs }),
});
