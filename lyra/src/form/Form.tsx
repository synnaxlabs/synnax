// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useMemo } from "react";
import { type z } from "zod";

import { Context, type ContextValue } from "@/form/Context";

/**
 * Publishes a form built by {@link use} to its subtree, so every field hook and
 * component below binds to it. Spread the hook's return value onto it.
 */
export const Form = <Z extends z.ZodType>({
  children,
  bind,
  set,
  get,
  mode,
  validate,
  validateAsync,
  value,
  has,
  remove,
  setStatus,
  clearStatuses,
  reset,
  setCurrentStateAsInitialValues,
  getStatuses,
}: PropsWithChildren<ContextValue<Z>>): ReactElement => {
  const ctx = useMemo(
    () =>
      ({
        bind,
        set,
        get,
        mode,
        validate,
        validateAsync,
        value,
        has,
        remove,
        setStatus,
        clearStatuses,
        reset,
        setCurrentStateAsInitialValues,
        getStatuses,
      }) as ContextValue,
    [
      bind,
      set,
      get,
      mode,
      validate,
      validateAsync,
      value,
      has,
      remove,
      setStatus,
      clearStatuses,
      reset,
      setCurrentStateAsInitialValues,
      getStatuses,
    ],
  );
  return <Context value={ctx}>{children}</Context>;
};
