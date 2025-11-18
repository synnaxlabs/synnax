// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { produce } from "immer";
import z from "zod";

export interface Action<Type extends string, Payload extends z.ZodType> {
  type: Type;
  payload: Payload;
}

export interface CreateActionReturn<
  State,
  Type extends string,
  Payload extends z.ZodType,
> {
  type: Type;
  payload: Payload;
  handler: (state: State, payload: z.infer<Payload>) => void;
  (payload: z.infer<Payload>): {
    type: Type;
    payload: z.infer<Payload>;
  };
}

export interface CreateActionParams<
  State,
  Type extends string,
  Payload extends z.ZodType,
> {
  type: Type;
  payload: Payload;
  handler: (state: State, payload: z.infer<Payload>) => void;
}

export const createAction = <State, Type extends string, Payload extends z.ZodType>(
  params: CreateActionParams<State, Type, Payload>,
): CreateActionReturn<State, Type, Payload> => {
  const action: CreateActionReturn<State, Type, Payload> = (
    payload: z.infer<Payload>,
  ) => ({
    type: params.type,
    payload,
  });
  action.type = params.type;
  action.payload = params.payload;
  action.handler = params.handler;
  return action;
};

export type InferAction<T extends CreateActionReturn<any, string, z.ZodType>> = {
  type: T["type"];
  payload: z.infer<T["payload"]>;
};

export type ActionsUnion<
  T extends readonly CreateActionReturn<any, string, z.ZodType>[],
> = {
  [K in keyof T]: T[K] extends CreateActionReturn<any, string, z.ZodType>
    ? InferAction<T[K]>
    : never;
}[number];

export type ActionConstructors<
  T extends readonly CreateActionReturn<any, string, z.ZodType>[],
> = {
  [K in T[number]["type"]]: (
    payload: z.infer<Extract<T[number], { type: K }>["payload"]>,
  ) => { type: K; payload: z.infer<Extract<T[number], { type: K }>["payload"]> };
};

export interface ReducerSystem<
  State,
  Actions extends readonly CreateActionReturn<State, string, z.ZodType>[],
> {
  reducer: (state: State, action: ActionsUnion<Actions>) => State;
  actionZ: z.ZodType<ActionsUnion<Actions>>;
}

export const createReducer = <
  State,
  const Actions extends readonly CreateActionReturn<State, string, z.ZodType>[],
>(
  actions: Actions,
): ReducerSystem<State, Actions> => {
  const handlerMap: Record<
    string,
    { schema: z.ZodType; handler: (state: State, payload: unknown) => void }
  > = {};

  for (const action of actions)
    handlerMap[action.type] = {
      schema: action.payload,
      handler: action.handler as (state: State, payload: unknown) => void,
    };

  const reducer = (state: State, action: { type: string; payload: unknown }): State => {
    const processor = handlerMap[action.type];
    if (processor == null) return state;
    const parsed = processor.schema.parse(action.payload);
    return produce(state, (draft) => processor.handler(draft as State, parsed));
  };

  const inputSchemas = actions.map((action) =>
    z
      .object({
        type: z.literal(action.type),
        payload: action.payload,
      })
      .transform((data) => ({
        type: data.type,
        [data.type]: data.payload,
      })),
  );

  const actionZ = z.union(
    inputSchemas as unknown as [z.ZodType<any>, z.ZodType<any>, ...z.ZodType<any>[]],
  ) as unknown as z.ZodType<ActionsUnion<Actions>>;

  return {
    reducer,
    actionZ,
  };
};
