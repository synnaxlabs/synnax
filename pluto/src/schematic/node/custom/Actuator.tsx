// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import { type MouseEventHandler, type ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { type Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { Symbol } from "@/schematic/symbol";

export interface ActuatorProps
  extends
    Omit<Toggle.ButtonProps, "onClick">,
    Pick<Primitive.SVGBasedProps, "strokeColor"> {
  specKey: string;
  onClick?: MouseEventHandler<HTMLElement>;
  scale?: number;
  stateOverrides?: schematic.symbol.State[];
}

export const Actuator = ({
  specKey,
  enabled = false,
  triggered = false,
  orientation = "left",
  scale = 1,
  className,
  stateOverrides,
  strokeColor,
  ...rest
}: ActuatorProps): ReactElement => {
  const { symbol, missing } = Symbol.useResolved(specKey);
  const spec = symbol?.data;
  const invalid = spec != null && !Custom.isValidSVG(spec.svg);
  if (missing || invalid)
    return (
      <Note.Note variant="warning" className={className}>
        <Text.Text level="p" status="warning">
          <Icon.Warning />
          {missing ? "Missing custom symbol" : "Invalid custom symbol"}
        </Text.Text>
      </Note.Note>
    );
  return (
    <Toggle.Button
      className={CSS.cls(
        CSS.BM("symbol", "custom"),
        CSS.B("custom-actuator"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        className,
      )}
      enabled={enabled}
      triggered={triggered}
      orientation={orientation}
      {...rest}
    >
      {spec != null && (
        <Custom.Drawing
          spec={spec}
          orientation={orientation}
          scale={scale}
          strokeColor={strokeColor}
          enabled={enabled}
          stateOverrides={stateOverrides}
        />
      )}
    </Toggle.Button>
  );
};
