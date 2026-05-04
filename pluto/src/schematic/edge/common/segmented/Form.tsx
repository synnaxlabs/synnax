// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";

export const Form = (): ReactElement => (
  <Flex.Box style={{ padding: "2rem" }} align="start" x>
    <Base.Field<color.Color> path="color" label="Color" padHelpText={false}>
      {({ value, onChange, variant: _, ...rest }) => (
        <Color.Swatch value={value} onChange={onChange} {...rest} />
      )}
    </Base.Field>
  </Flex.Box>
);
