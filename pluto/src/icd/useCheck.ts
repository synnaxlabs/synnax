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

export interface UseCheckArgs {
  /** The form that edits the ICD. */
  form: Pick<Form.ContextValue, "bind" | "setStatus"> & { value: () => icd.New };
}

/**
 * Checks the ICD in the form when it stays unchanged for {@link CHECK_DELAY} after an
 * edit, and puts each problem the check finds on the form path it names. An edit clears
 * the problems until the next check.
 */
export const useCheck = ({ form }: UseCheckArgs): void => {
  const client = Synnax.use();
  const { bind, value, setStatus } = form;
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (client == null) return;
    let checked: icd.New | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const run = () =>
      handleError(async () => {
        const v = deep.copy(value());
        checked = v;
        const problems = await client.icds.check(v);
        // The form changed while the check ran, so its problems are out of date.
        if (!deep.equal(value(), v)) return;
        markProblems(problems, v, setStatus);
      }, "Failed to check");
    const unbind = bind(() => {
      if (deep.equal(value(), checked)) return;
      clearTimeout(timer);
      timer = setTimeout(run, CHECK_DELAY.milliseconds);
    });
    return () => {
      clearTimeout(timer);
      unbind();
    };
  }, [client, bind, value, setStatus, handleError]);
};
