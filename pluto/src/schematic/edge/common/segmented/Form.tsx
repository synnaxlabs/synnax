// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/common/segmented/Segmented.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Color } from "@/color";
import { type Variant } from "@/schematic/edge/registry";

const SELECT_STYLE: CSSProperties = { width: "25rem" };

interface SelectVariantProps extends Omit<
  Select.SingleSimpleProps<Variant>,
  "children" | "resourceName"
> {}

const SelectVariant = (props: SelectVariantProps): ReactElement => (
  <Select.Simple<Variant> {...props} resourceName="edge type" style={SELECT_STYLE}>
    <Select.Item itemKey="pipe">Pipe</Select.Item>
    <Select.Item itemKey="electric">Electrical</Select.Item>
    <Select.Item itemKey="secondary">Secondary</Select.Item>
    <Select.Item itemKey="jacketed">Jacketed</Select.Item>
    <Select.Item itemKey="hydraulic">Hydraulic</Select.Item>
    <Select.Item itemKey="pneumatic">Pneumatic</Select.Item>
    <Select.Item itemKey="data">Data</Select.Item>
  </Select.Simple>
);

export const Form = (): ReactElement => (
  <Flex.Box className={CSS.B("schematic-edge-form")} align="start" x>
    <Base.Field<color.Color> path="color" label="Color" padHelpText={false}>
      {(p) => <Color.Swatch {...p} />}
    </Base.Field>
    <Base.Field<Variant> path="variant" label="Variant" padHelpText={false}>
      {(p) => <SelectVariant {...p} />}
    </Base.Field>
  </Flex.Box>
);
