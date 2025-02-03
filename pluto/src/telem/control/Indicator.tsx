// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/telem/control/Indicator.css";

import { TimeStamp } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Color } from "@/color";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/memo";
import { control } from "@/telem/control/aether";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";

export interface IndicatorProps
  extends Omit<z.input<typeof control.indicatorStateZ>, "status" | "color">,
    PropsWithChildren {}

export const Indicator = ({
  colorSource,
  statusSource,
}: IndicatorProps): ReactElement => {
  const memoProps = useMemoDeepEqualProps({ colorSource, statusSource });

  const [, { color, status }, setState] = Aether.use({
    type: control.Indicator.TYPE,
    initialState: {
      ...memoProps,
      status: {
        key: "no_chip",
        variant: "warning",
        message: "No chip connected.",
        time: TimeStamp.now(),
      },
    },
    schema: control.indicatorStateZ,
  });

  useEffect(() => {
    setState((p) => ({ ...p, ...memoProps }));
  }, [memoProps, setState]);

  let parsedColor: Color.Crude;
  if (status.data?.color != null) parsedColor = Color.Color.z.parse(status.data.color);
  else if (color != null && !color.isZero) parsedColor = color;
  else parsedColor = "var(--pluto-gray-l8)";

  return (
    <Tooltip.Dialog location={{ x: "center", y: "bottom" }}>
      <Text.Text level="p">{status.message}</Text.Text>
      <div
        className={CSS.B("indicator")}
        style={{
          backgroundColor: Color.cssString(parsedColor),
          flexGrow: 1,
        }}
      />
    </Tooltip.Dialog>
  );
};
