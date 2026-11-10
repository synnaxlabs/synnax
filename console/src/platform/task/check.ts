// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** Reports an issue at a path in the checked value. A warning never blocks a deploy. */
export interface ReportIssue {
  (path: PropertyKey[], message: string, variant?: "warning"): void;
}

/**
 * A schema that passes every value and reports issues through `report`, for a check
 * that the form runs with `Form.useCheck`.
 */
export const check = <T>(fn: (value: T, report: ReportIssue) => void): z.ZodType<T> =>
  z.custom<T>().check(({ value, issues }) =>
    fn(value, (path, message, variant) =>
      issues.push({
        code: "custom",
        path,
        message,
        input: value,
        params: variant == null ? undefined : { variant },
      }),
    ),
  );
