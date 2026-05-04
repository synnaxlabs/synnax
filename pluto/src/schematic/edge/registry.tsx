// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";
import z from "zod";

import { dataSpec } from "@/schematic/edge/Data";
import { electricSpec } from "@/schematic/edge/Electric";
import { hydraulicSpec } from "@/schematic/edge/Hydraulic";
import { jacketedSpec } from "@/schematic/edge/Jacketed";
import { pipeSpec } from "@/schematic/edge/Pipe";
import { pneumaticSpec } from "@/schematic/edge/Pneumatic";
import { secondarySpec } from "@/schematic/edge/Secondary";
import { type Edge, type Spec } from "@/schematic/edge/spec";
import { Select } from "@/select";

const REGISTRY = {
  pipe: pipeSpec,
  electric: electricSpec,
  secondary: secondarySpec,
  jacketed: jacketedSpec,
  hydraulic: hydraulicSpec,
  pneumatic: pneumaticSpec,
  data: dataSpec,
} as const;

const VARIANTS = Object.keys(REGISTRY);
export const variantZ = z.enum(VARIANTS);
export type Variant = keyof typeof REGISTRY;
export type Config = ReturnType<
  (typeof REGISTRY)[keyof typeof REGISTRY]["defaultConfig"]
>;

const SELECT_DATA: record.KeyedNamed<Variant>[] = Object.values(REGISTRY).map(
  ({ variant, name }) => ({ key: variant, name }),
);

const SELECT_STYLE: CSSProperties = { width: "25rem" };

export interface SelectEdgeTypeProps extends Omit<
  Select.StaticProps<Variant>,
  "data" | "resourceName"
> {}

export const SelectEdgeType = (props: SelectEdgeTypeProps): ReactElement => (
  <Select.Static
    {...props}
    data={SELECT_DATA}
    resourceName="edge type"
    style={{ ...SELECT_STYLE, ...props.style }}
  />
);

const resolveSpec = (variant: string): Spec<Variant, any> => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null) throw new NotFoundError(`Edge with variant ${variant} not found`);
  return spec;
};

export const resolve = (variant: string): Edge => resolveSpec(variant).Edge;
