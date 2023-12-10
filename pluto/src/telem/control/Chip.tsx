// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Optional } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { Status } from "@/status";
import { Text } from "@/text";
import { Toggle } from "@/vis/toggle";

export interface ChipProps
  extends Optional<Toggle.UseProps, "aetherKey">,
    Omit<Button.IconProps, "onClick" | "children"> {}

export const Chip = Aether.wrap<ChipProps>(
  "Chip",
  ({ aetherKey, source, sink, ...props }): ReactElement => {
    const { enabled, toggle } = Toggle.use({ aetherKey, source, sink });

    return (
      <Button.Icon
        variant="text"
        onClick={toggle}
        tooltip={
          <Text.Text level="small">
            {enabled
              ? "Absolute control. Click to release."
              : "Click to take absolute control."}
          </Text.Text>
        }
        {...props}
      >
        <Status.Circle variant={enabled ? "success" : "error"} />
      </Button.Icon>
    );
  },
);
