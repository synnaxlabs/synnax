// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status as Base } from "@synnaxlabs/lyra/status";
import { Status } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { getSelected } from "@/platform/status/filter/selected";
import { View } from "@/platform/view";

const decode = (node: status.Where): status.Variant[] | null => {
  if (View.Filter.soleField(node) !== "variant") return null;
  const variants = getSelected(node.variant);
  return variants.length === 0 ? null : variants;
};

const Picker = ({
  value,
  onChange,
}: View.Filter.PickerProps<status.Variant[]>): ReactElement => (
  <Select.Frame<status.Variant, undefined>
    multiple
    value={value ?? []}
    onChange={(next: status.Variant[]) => onChange(next.length === 0 ? null : next)}
  >
    <Select.Dialog surface="menu" aria-label="Variants">
      <Select.List>{Status.VARIANT_ITEMS}</Select.List>
    </Select.Dialog>
  </Select.Frame>
);

const Operator = ({
  value,
}: View.Filter.ConditionProps<status.Variant[]>): ReactElement => (
  <View.Filter.ChipOperator>
    {value.length === 1 ? "is" : "is any of"}
  </View.Filter.ChipOperator>
);

const Value = ({
  value,
}: View.Filter.ConditionProps<status.Variant[]>): ReactElement => (
  <>
    {value.map((variant) => (
      <Base.Indicator key={variant} variant={variant} />
    ))}
    {value.map((variant) => caseconv.capitalize(variant)).join(", ")}
  </>
);

/** The variant property of a view filter. */
export const VARIANT: View.Filter.Property<status.Where, status.Variant[]> = {
  name: "Variants",
  icon: Icon.Status,
  decode,
  encode: (variants) => ({ variant: { in: variants } }),
  Picker,
  Operator,
  Value,
};
