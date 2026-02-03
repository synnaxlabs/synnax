// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/telem/control/Indicator.css";

import { color, TimeStamp } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useMemoDeepEqual } from "@/memo";
import { control } from "@/telem/control/aether";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";

export interface IndicatorProps
  extends
    Omit<z.input<typeof control.indicatorStateZ>, "status" | "color">,
    PropsWithChildren {}

export interface StatusDetails {
  color?: string;
}

export const Indicator = ({
  colorSource,
  statusSource,
}: IndicatorProps): ReactElement => {
  const memoProps = useMemoDeepEqual({ colorSource, statusSource });

  const [, { color: colorVal, status }, setState] = Aether.use({
    type: control.Indicator.TYPE,
    initialState: {
      ...memoProps,
      status: {
        key: "no_chip",
        variant: "warning",
        message: "No chip connected.",
        time: TimeStamp.now(),
        details: { color: undefined },
      },
    },
    schema: control.indicatorStateZ,
  });

  useEffect(() => {
    setState((p) => ({ ...p, ...memoProps }));
  }, [memoProps, setState]);

  let parsedColor: color.Crude;
  if (status.details?.color != null)
    parsedColor = color.colorZ.parse(status.details.color);
  else if (colorVal != null && !color.isZero(colorVal)) parsedColor = colorVal;
  else parsedColor = "var(--pluto-gray-l10)";

  return (
    <Tooltip.Dialog location={{ x: "center", y: "bottom" }}>
      <Text.Text>{status.message}</Text.Text>
      <div
        className={CSS.B("indicator")}
        style={{
          backgroundColor: color.cssString(parsedColor),
          flexGrow: 1,
        }}
      />
    </Tooltip.Dialog>
  );
};
