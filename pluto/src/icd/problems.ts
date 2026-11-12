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
import { deep } from "@synnaxlabs/x";

/**
 * Rewrites a dotted problem path, such as "messages.2.bitLength", into the form path of
 * the value it names, with element keys in place of array indexes.
 */
const toFormPath = (path: string, value: unknown): deep.Target<unknown> =>
  deep
    .resolvePath(path, value)
    .split(".")
    .reduce<deep.Path<unknown>>((target, part) => target[part], deep.path<unknown>());

/**
 * Puts each problem as an error status on the form path it names in the value.
 * @returns the form paths it marked.
 */
export const markProblems = (
  problems: icd.Problem[],
  value: unknown,
  setStatus: Form.ContextValue["setStatus"],
): deep.Target<unknown>[] =>
  problems.map(({ path, message }) => {
    const formPath = toFormPath(path, value);
    setStatus(formPath, { key: String(formPath), variant: "error", message });
    return formPath;
  });
