// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { type Form } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";
import { deep, TimeSpan } from "@synnaxlabs/x";
import { useEffect } from "react";

import { markProblems } from "@/icd/problems";
import { Synnax } from "@/synnax";

/** How long the form must stay unchanged before a check runs. */
export const CHECK_DELAY = TimeSpan.milliseconds(500);

export interface UseCheckParams {
  /** The key of the ICD that the form edits. */
  key: icd.Key;
  /** The form that edits the ICD. */
  form: Pick<Form.ContextValue, "bind" | "setStatus" | "clearStatus"> & {
    value: () => icd.New;
  };
}

/**
 * Checks the ICD in the form when the hook mounts, and again when the form stays
 * unchanged for {@link CHECK_DELAY} after an edit. Puts each problem a check finds on
 * the form path it names, and clears the problems the previous check found. An edit
 * clears the problem on the edited path until the next check.
 */
export const useCheck = ({ key, form }: UseCheckParams): void => {
  const client = Synnax.use();
  const { bind, value, setStatus, clearStatus } = form;
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (client == null) return;
    let checked: icd.New | null = null;
    let marked: deep.Target<unknown>[] = [];
    let timer: ReturnType<typeof setTimeout> | undefined;
    const run = () =>
      handleError(async () => {
        const v = deep.copy(value());
        checked = v;
        const problems = await client.icds.check({ ...v, key });
        // The form changed while the check ran, so its problems are out of date.
        if (!deep.equal(value(), v)) return;
        marked.forEach(clearStatus);
        marked = markProblems(problems, v, setStatus);
      }, "Failed to check");
    run();
    const unbind = bind(() => {
      if (deep.equal(value(), checked)) return;
      clearTimeout(timer);
      timer = setTimeout(run, CHECK_DELAY.milliseconds);
    });
    return () => {
      clearTimeout(timer);
      unbind();
    };
  }, [client, key, bind, value, setStatus, clearStatus, handleError]);
};
