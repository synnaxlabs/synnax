// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Select } from "@/select";
import { Button, Buttons } from "@/select/Button";
import { text } from "@/text/core";

export interface LevelProps extends Omit<Select.ButtonsProps<text.Level>, "keys"> {}

export const Level = (props: LevelProps): ReactElement => (
  <Buttons {...props} keys={text.LEVELS}>
    <Button itemKey="h2" square>
      XL
    </Button>
    <Button itemKey="h3" square>
      L
    </Button>
    <Button itemKey="h4" square>
      M
    </Button>
    <Button itemKey="h5" square>
      S
    </Button>
    <Button itemKey="small" square>
      XS
    </Button>
  </Buttons>
);
