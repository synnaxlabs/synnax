import type { Action, AnyAction } from "@reduxjs/toolkit";

import { Event } from "@/runtime";
import { StoreState } from "@/state";

export const encode = <S extends StoreState, A extends Action = AnyAction>(
  event: Event<S, A>
): string => JSON.stringify(event);
export const decode = <S extends StoreState, A extends Action = AnyAction>(
  event: string
): Event<S, A> => JSON.parse(event) as Event<S, A>;
