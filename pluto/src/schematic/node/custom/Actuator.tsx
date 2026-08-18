// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type CrudeTimeSpan } from "@synnaxlabs/x";
import { type MouseEventHandler, type ReactElement } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Note } from "@/note";
import { Custom } from "@/schematic/node/common/custom";
import { Handle } from "@/schematic/node/common/handle";
import { Toggle } from "@/schematic/node/common/toggle";
import { Symbol } from "@/schematic/symbol";
import { Text } from "@/text";

export interface ActuatorProps extends Omit<Toggle.ButtonProps, "onClick"> {
  specKey: string;
  onClick?: MouseEventHandler<HTMLElement>;
  onClickDelay?: CrudeTimeSpan;
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
  ...rest
}: ActuatorProps): ReactElement => {
  const { symbol, missing } = Symbol.useResolved(specKey);
  const spec = symbol?.data;
  const setContainer = Custom.useRender({
    orientation,
    activeState: enabled ? "active" : "base",
    externalScale: scale,
    spec,
    stateOverrides,
  });
  if (missing)
    return (
      <Note.Note variant="warning" className={className}>
        <Text.Text level="p" status="warning">
          <Icon.Warning />
          Missing custom symbol
        </Text.Text>
      </Note.Note>
    );
  const handles = spec?.handles ?? [];
  return (
    <Toggle.Button
      ref={setContainer}
      className={CSS.cx(
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
      <Handle.Boundary orientation={orientation}>
        {handles.map((handle) => (
          <Handle.Handle
            key={handle.key}
            id={handle.key}
            location={handle.orientation}
            orientation={orientation}
            left={handle.position.x * 100}
            top={handle.position.y * 100}
          />
        ))}
      </Handle.Boundary>
    </Toggle.Button>
  );
};
