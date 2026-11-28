// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { z } from "zod";

import { strings } from "@/strings";

/**
 * Words a Zod issue as a sentence that names its field, as the Core words the problems
 * with its schemas, such as "port is required". The field is the last string key of the
 * issue's path in sentence case, or "value" when the path has none. As the error map of
 * a parse, it words only the issues whose schema gives no message of its own.
 */
export const issueMessage = (issue: z.core.$ZodIssue | z.core.$ZodRawIssue): string => {
  const name = issue.path?.findLast((p) => typeof p === "string") ?? "";
  const field = name === "" ? "value" : strings.toSentence(name).toLowerCase();
  if (issue.code === "too_small") {
    if (issue.origin === "string")
      return Number(issue.minimum) <= 1
        ? `${field} is required`
        : `${field} must be at least ${issue.minimum} characters long`;
    const bound =
      issue.inclusive === false ? "greater than" : "greater than or equal to";
    return `${field} must be ${bound} ${issue.minimum}`;
  }
  if (issue.code === "too_big") {
    if (issue.origin === "string")
      return `${field} must be at most ${issue.maximum} characters long`;
    const bound = issue.inclusive === false ? "less than" : "less than or equal to";
    return `${field} must be ${bound} ${issue.maximum}`;
  }
  return `${field} is not valid`;
};
