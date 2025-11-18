import { type Destructor } from "@synnaxlabs/x";
import { produce } from "immer";
import { useCallback, useReducer, useRef, useState } from "react";

import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
import { errorResult, loadingResult, type Result, successResult } from "@/flux/result";
import { type RetrieveParams } from "@/flux/retrieve";
import { useAsyncEffect, useDestructors } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { Synnax } from "@/synnax";

export interface Action<
  Payload extends core.Shape = core.Shape,
  Type extends string = string,
> {
  type: Type;
  payload: Payload;
}

export interface ActionHandler<Payload extends core.Shape, State extends core.Shape> {
  (state: State, payload: Payload): State | void;
}

const SET_STATE_ACTION = "__setState__" as const;

interface SetStateAction<State extends core.Shape>
  extends Action<State, typeof SET_STATE_ACTION> {}

type ActionsToUnion<
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> = {
  [K in keyof Actions]: Actions[K] extends ActionHandler<infer P, State>
    ? Action<P, K & string>
    : never;
}[keyof Actions];

type AllActions<
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> = ActionsToUnion<State, Actions> | SetStateAction<State>;

export interface Dispatch<
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> {
  (action: ActionsToUnion<State, Actions>): void;
}

export interface MountModelListenersParams<
  Query extends core.Shape,
  Store extends core.Store,
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> extends RetrieveParams<Query, Store> {
  dispatch: Dispatch<State, Actions>;
}

export interface UpdateModelParams<
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> {
  action: ActionsToUnion<State, Actions>;
}

export interface CreateModelParams<
  Query extends core.Shape,
  State extends core.Shape,
  Store extends core.Store,
  Actions extends Record<string, ActionHandler<any, State>>,
> {
  name: string;
  retrieve: (params: RetrieveParams<Query, Store>) => Promise<State>;
  update: (params: UpdateModelParams<State, Actions>) => Promise<void>;
  actions: Actions;
  initialState: State;
  mountListeners?: (
    params: MountModelListenersParams<Query, Store, State, Actions>,
  ) => Destructor | Destructor[];
}

export interface UseModelParams<Query extends core.Shape> {
  query: Query;
}

export interface UseModelReturn<
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> {
  result: Result<State>;
  dispatch: Dispatch<State, Actions>;
}

export interface UseModel<
  Query extends core.Shape,
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> {
  (params: UseModelParams<Query>): UseModelReturn<State, Actions>;
}

const useModelInternal = <
  Query extends core.Shape,
  State extends core.Shape,
  Store extends core.Store,
  Actions extends Record<string, ActionHandler<any, State>>,
>({
  name,
  retrieve,
  actions,
  initialState,
  mountListeners,
  query,
}: UseModelParams<Query> &
  CreateModelParams<Query, State, Store, Actions>): UseModelReturn<State, Actions> => {
  const client = Synnax.use();
  const queryRef = useRef<Query>(query);
  const store = useStore<Store>();
  const [result, setResult] = useState<Omit<Result<State>, "data">>(
    loadingResult<State>(`retrieving ${name}`),
  );
  const [currState, internalDispatch] = useReducer<State, [AllActions<State, Actions>]>(
    (state, action): State => {
      if (action.type === SET_STATE_ACTION) return action.payload as State;
      if (action.type in actions)
        return produce(state, (draft: any) =>
          actions[action.type](draft as State, action.payload),
        );

      return state;
    },
    initialState,
  );

  const setState = useCallback(
    (state: State) => {
      internalDispatch({ type: SET_STATE_ACTION, payload: state });
    },
    [internalDispatch],
  );

  const dispatch = useCallback(
    (action: ActionsToUnion<State, Actions>) => {
      internalDispatch(action);
    },
    [internalDispatch],
  );

  const listeners = useDestructors();

  const retrieveAsync = useCallback(
    async (
      querySetter: state.SetArg<Query, Partial<Query>>,
      options: core.FetchOptions = {},
    ) => {
      const { signal } = options;
      const query = state.executeSetter<Query, Partial<Query>>(
        querySetter,
        queryRef.current ?? {},
      );
      if (client == null) return;
      const params: RetrieveParams<Query, Store> = { client, query, store };
      try {
        setResult(loadingResult(`retrieving ${name}`));
        const value = await retrieve(params);
        if (signal?.aborted) return;
        listeners.cleanup();
        listeners.set(mountListeners?.({ ...params, dispatch }));
        setState(value);
        setResult(successResult(`retrieved ${name}`, value));
      } catch (error) {
        setResult(errorResult(`retrieve ${name}`, error));
      }
    },
    [client, store, retrieve, name, setState, dispatch, mountListeners],
  );
  const memoQuery = useMemoDeepEqual(queryRef.current);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoQuery, { signal }),
    [retrieveAsync, memoQuery],
  );
  return {
    result: { ...result, data: currState } as Result<State>,
    dispatch,
  };
};

export interface CreateModelReturn<
  Query extends core.Shape,
  State extends core.Shape,
  Actions extends Record<string, ActionHandler<any, State>>,
> {
  useModel: UseModel<Query, State, Actions>;
}

export const createModel = <
  Query extends core.Shape,
  State extends core.Shape,
  Store extends core.Store,
  Actions extends Record<string, ActionHandler<any, State>>,
>(
  createParams: CreateModelParams<Query, State, Store, Actions>,
): CreateModelReturn<Query, State, Actions> => ({
  useModel: (params: UseModelParams<Query>) =>
    useModelInternal({ ...params, ...createParams }),
});
