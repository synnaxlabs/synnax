// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";

import { type Condition, decode, encode } from "@/platform/view/filter/codec";
import { type Properties } from "@/platform/view/filter/types";

/**
 * @returns the conditions of the filter of the view in the form, and a setter that
 * writes new conditions back to the filter.
 */
export const useConditions = <W extends { and?: W[] }>(
  properties: Properties<W>,
): [Condition<W>[], (next: Condition<W>[]) => void] => {
  const path = deep.path<{ where?: W }>();
  const where = Form.useFieldValue<W | undefined>(path.where, { optional: true });
  const { set } = Form.useContext();
  return [
    decode(where ?? undefined, properties),
    (next) => set(path.where, encode(next, properties)),
  ];
};
