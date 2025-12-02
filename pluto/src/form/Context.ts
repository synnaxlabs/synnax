// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type status } from "@synnaxlabs/x";
import { type z } from "zod";

import { context } from "@/context";
import { type State } from "@/form/state";

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
  (props: Listener): destructor.Destructor;
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

const [Context, useCtx] = context.create<ContextValue | null>({
  defaultValue: null,
  displayName: "Form.Context",
});
export { Context };

export const useContext = <Z extends z.ZodType = z.ZodType>(
  override?: ContextValue<Z>,
  funcName: string = "Form.useContext",
): ContextValue<Z> => {
  const internal = useCtx();
  if (internal == null && override == null)
    throw new Error(`${funcName} must be used within a Form context value`);
  return override ?? (internal as unknown as ContextValue<Z>);
};
