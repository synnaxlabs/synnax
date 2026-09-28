// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Select } from "@/select";
import { Buttons } from "@/select/Buttons";
import { Item } from "@/select/Item";

export interface LevelProps extends Select.ButtonsProps<text.Level> {}

export const Level = (props: LevelProps): ReactElement => (
  <Buttons {...props}>
    <Item itemKey="small" square>
      XS
    </Item>
    <Item itemKey="h5" square>
      S
    </Item>
    <Item itemKey="h4" square>
      M
    </Item>
    <Item itemKey="h3" square>
      L
    </Item>
    <Item itemKey="h2" square>
      XL
    </Item>
  </Buttons>
);
