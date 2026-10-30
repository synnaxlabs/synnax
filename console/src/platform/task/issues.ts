// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status } from "@synnaxlabs/lyra/status";
import { deep } from "@synnaxlabs/x";
import { type z } from "zod";

const variantOf = (issue: z.core.$ZodIssue): Status.Variant =>
  issue.code === "custom" && issue.params != null && "variant" in issue.params
    ? (issue.params.variant as Status.Variant)
    : "error";

/**
 * The field statuses of a check's issues on the config, keyed by their path in the
 * form. An issue that carries a `variant` param takes it; any other is an error. The
 * first issue on a path wins, but an error outranks a warning.
 */
export const issueStatuses = (
  issues: z.core.$ZodIssue[],
  values: unknown,
): Map<string, Status.Crude> => {
  const statuses = new Map<string, Status.Crude>();
  issues.forEach((issue) => {
    // Fields address array items by their key, not by their index.
    const path = deep.resolvePath(["config", ...issue.path].join("."), values);
    const variant = variantOf(issue);
    const prev = statuses.get(path);
    if (prev != null && (variant === "warning" || prev.variant === "error")) return;
    statuses.set(path, { key: path, variant, message: issue.message });
  });
  return statuses;
};
