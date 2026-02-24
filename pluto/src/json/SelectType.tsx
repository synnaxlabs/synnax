// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type PrimitiveTypeName } from "@/json/primitive";
import { Select } from "@/select";

const KEYS: PrimitiveTypeName[] = ["string", "number", "boolean", "null"];

export interface SelectTypeProps extends optional.Optional<
  Select.ButtonsProps<PrimitiveTypeName>,
  "keys"
> {}

export const SelectType = (props: SelectTypeProps): ReactElement => (
  <Select.Buttons<PrimitiveTypeName> keys={KEYS} {...props}>
    <Select.Button itemKey="string">String</Select.Button>
    <Select.Button itemKey="number">Number</Select.Button>
    <Select.Button itemKey="boolean">Boolean</Select.Button>
    <Select.Button itemKey="null">Null</Select.Button>
  </Select.Buttons>
);
