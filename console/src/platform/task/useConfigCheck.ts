// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type Status } from "@synnaxlabs/lyra/status";
import { useEffect, useRef } from "react";
import { type z } from "zod";

import { issueStatuses } from "@/platform/task/issues";

const CLEAR: Status.Crude = { variant: "success", message: "" };

/**
 * Shows the issues of a check on the task config as field statuses while the user
 * edits, as a deploy shows them. A null check shows none.
 */
export const useConfigCheck = (check: z.ZodType | null): void => {
  const ctx = Form.useContext();
  const config = Form.useFieldValue<unknown>("config");
  const shown = useRef<string[]>([]);
  useEffect(() => {
    const result = check?.safeParse(config);
    const next =
      result == null || result.success
        ? new Map<string, Status.Crude>()
        : issueStatuses(result.error.issues, ctx.value());
    const set = (path: string, status: Status.Crude) => {
      const current = ctx.get(path, { optional: true })?.status;
      if (current?.variant === status.variant && current.message === status.message)
        return;
      ctx.setStatus(path, status);
    };
    shown.current.forEach((path) => {
      if (!next.has(path) && ctx.get(path, { optional: true }) != null)
        set(path, CLEAR);
    });
    next.forEach((status, path) => set(path, status));
    shown.current = [...next.keys()];
  }, [check, config]);
};
