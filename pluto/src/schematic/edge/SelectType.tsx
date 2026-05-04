// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { REGISTRY, type Variant } from "@/schematic/edge/registry";
import { Select } from "@/select";

const SELECT_DATA: record.KeyedNamed<Variant>[] = Object.values(REGISTRY).map(
  ({ key, name }) => ({ key, name }),
);

const SELECT_STYLE: CSSProperties = { width: "25rem" };

export interface SelectVariantProps extends Omit<
  Select.StaticProps<Variant>,
  "data" | "resourceName"
> {}

export const SelectVariant = (props: SelectVariantProps): ReactElement => (
  <Select.Static
    {...props}
    data={SELECT_DATA}
    resourceName="edge type"
    style={{ ...SELECT_STYLE, ...props.style }}
  />
);
