// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Select } from "@/select";
import { text } from "@/text/core";

export interface LevelProps extends Select.SingleProps<text.Level> {}

export const Level = ({ value, onChange, ...rest }: LevelProps): ReactElement => (
  <Select.Buttons {...rest} keys={text.LEVELS} value={value} onChange={onChange}>
    <Select.Button itemKey="h2">XL</Select.Button>
    <Select.Button itemKey="h3">L</Select.Button>
    <Select.Button itemKey="h4">M</Select.Button>
    <Select.Button itemKey="h5">S</Select.Button>
    <Select.Button itemKey="small">XS</Select.Button>
  </Select.Buttons>
);
