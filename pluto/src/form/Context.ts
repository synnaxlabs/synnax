// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor, type status } from "@synnaxlabs/x";
import { createContext, use } from "react";
import { type z } from "zod";

import { type FieldState, type State } from "@/form/state";

export interface RemoveFunc {
  (path: string): void;
}

export interface SetOptions {
  notifyOnChange?: boolean;
  markTouched?: boolean;
}

export interface SetFunc {
  (path: string, value: unknown, options?: SetOptions): void;
}

export interface Listener {
  (): void;
}

export interface BindFunc {
  (props: Listener): Destructor;
}

export type Mode = "normal" | "preview";
export interface ContextValue<Z extends z.ZodType = z.ZodType> {
  mode: Mode;
  bind: BindFunc;
  set: SetFunc;
  get: typeof State.prototype.getState;
  reset: (values?: z.infer<Z>) => void;
  remove: RemoveFunc;
  value: () => z.infer<Z>;
  validate: (path?: string) => boolean;
  validateAsync: (path?: string) => Promise<boolean>;
  has: (path: string) => boolean;
  setStatus: typeof State.prototype.setStatus;
  clearStatuses: () => void;
  setCurrentStateAsInitialValues: () => void;
  getStatuses: () => status.Crude[];
}

export const Context = createContext<ContextValue>({
  mode: "normal",
  bind: () => () => {},
  set: () => {},
  reset: () => {},
  remove: () => {},
  get: <V = unknown>(): FieldState<V> => ({
    value: undefined as V,
    status: { key: "", variant: "success", message: "" },
    touched: false,
    required: false,
  }),
  validate: () => false,
  validateAsync: () => Promise.resolve(false),
  value: () => ({}),
  has: () => false,
  setStatus: () => {},
  clearStatuses: () => {},
  setCurrentStateAsInitialValues: () => {},
  getStatuses: () => [],
});

export const useContext = <Z extends z.ZodType = z.ZodType>(
  override?: ContextValue<Z>,
): ContextValue<Z> => {
  const internal = use(Context);
  return override ?? (internal as unknown as ContextValue<Z>);
};
