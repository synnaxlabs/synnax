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
import { type text } from "@/text/core";

const DATA = [250, 400, 500, 600];

export interface WeightProps extends Omit<Select.ButtonsProps<text.Weight>, "keys"> {}

export const Weight = (props: WeightProps): ReactElement => (
  <Buttons {...props} keys={DATA}>
    <Button itemKey={250}>Light</Button>
    <Button itemKey={400}>Normal</Button>
    <Button itemKey={500}>Medium</Button>
    <Button itemKey={600}>Bold</Button>
  </Buttons>
);
