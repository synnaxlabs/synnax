import { StoreState } from "@/state";
import { Action, AnyAction } from "@reduxjs/toolkit";
import { Event } from "@/runtime";

export const encode = <S extends StoreState, A extends Action = AnyAction>(
  event: Event<S, A>
) => JSON.stringify(event);
export const decode = <S extends StoreState, A extends Action = AnyAction>(
  event: string
) => JSON.parse(event) as Event<S, A>;
