// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { caseconv, deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { getSelected } from "@/platform/status/filter/selected";
import { View } from "@/platform/view";

const path = deep.path<{ where?: { variant?: status.Where["variant"] } }>();

/** A chip for the variants in the filter of the view. Without any, it is absent. */
export const Chips = (): ReactElement | null => {
  const field = Form.useField(deep.present(path.where).variant, {
    optional: true,
  });
  const variants = getSelected(field?.value);
  if (variants.length === 0 || field == null) return null;
  return (
    <View.FilterChip>
      <View.FilterChipName icon={Icon.Status}>Variants</View.FilterChipName>
      <View.FilterChipOperator>
        {variants.length === 1 ? "is" : "is any of"}
      </View.FilterChipOperator>
      <View.FilterChipValue>
        {variants.map((variant) => (
          <Status.Indicator key={variant} variant={variant} />
        ))}
        {variants.map((variant) => caseconv.capitalize(variant)).join(", ")}
      </View.FilterChipValue>
      <View.FilterChipRemove
        name="Variants"
        onClick={() => field.onChange(undefined)}
      />
    </View.FilterChip>
  );
};
