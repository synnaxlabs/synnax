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

export interface LevelProps extends Omit<Select.ButtonsProps<text.Level>, "keys"> {}

export const Level = (props: LevelProps): ReactElement => (
  <Select.Buttons {...props} keys={text.LEVELS}>
    <Select.Button itemKey="h2" square>
      XL
    </Select.Button>
    <Select.Button itemKey="h3" square>
      L
    </Select.Button>
    <Select.Button itemKey="h4" square>
      M
    </Select.Button>
    <Select.Button itemKey="h5" square>
      S
    </Select.Button>
    <Select.Button itemKey="small" square>
      XS
    </Select.Button>
  </Select.Buttons>
);
