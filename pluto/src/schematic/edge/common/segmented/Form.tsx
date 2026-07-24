// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/common/segmented/Segmented.css";

import { type color, type record } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { type Variant } from "@/schematic/edge/registry";
import { Select } from "@/select";

const SELECT_DATA: record.KeyedNamed<Variant>[] = [
  { key: "pipe", name: "Pipe" },
  { key: "electric", name: "Electrical" },
  { key: "secondary", name: "Secondary" },
  { key: "jacketed", name: "Jacketed" },
  { key: "hydraulic", name: "Hydraulic" },
  { key: "pneumatic", name: "Pneumatic" },
  { key: "data", name: "Data" },
];

const SELECT_STYLE: CSSProperties = { width: "25rem" };

interface SelectVariantProps extends Omit<
  Select.StaticProps<Variant>,
  "data" | "resourceName"
> {}

const SelectVariant = (props: SelectVariantProps): ReactElement => (
  <Select.Static
    {...props}
    data={SELECT_DATA}
    resourceName="edge type"
    style={SELECT_STYLE}
  />
);

export const Form = (): ReactElement => (
  <Flex.Box className={CSS.B("schematic-edge-form")} align="start" x>
    <Base.Field<color.Color> path="color" label="Color" padHelpText={false}>
      {({ value, onChange, variant: _, ...rest }) => (
        <Color.Swatch value={value} onChange={onChange} {...rest} />
      )}
    </Base.Field>
    <Base.Field<Variant> path="variant" label="Variant" padHelpText={false}>
      {({ value, onChange, variant: _, ...rest }) => (
        <SelectVariant value={value} onChange={onChange} {...rest} />
      )}
    </Base.Field>
  </Flex.Box>
);
