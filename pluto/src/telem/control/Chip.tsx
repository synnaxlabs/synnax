// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control as clientControl } from "@synnaxlabs/client";
import { type status, TimeStamp } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, useCallback, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { useMemoDeepEqual } from "@/memo";
import { control } from "@/telem/control/aether";
import { Text } from "@/text";

export interface ChipProps
  extends
    Pick<z.input<typeof control.chipStateZ>, "source" | "sink">,
    Omit<Button.ButtonProps, "onClick" | "children"> {}

interface ChipStyle {
  message: string;
  chipColor: string;
  buttonStyle?: CSSProperties;
  disabled?: boolean;
}

const tooltipMessage = (
  status: status.Status<typeof control.chipStatusDetailsZ>,
): ChipStyle => {
  switch (status.variant) {
    case "disabled":
      if (status.details?.valid === true)
        return {
          message: "Uncontrolled. Click to take control.",
          chipColor: "var(--pluto-gray-l12)",
        };
      return {
        message: "No channel connected. This element cannot be controlled.",
        chipColor: "var(--pluto-gray-l7)",
        disabled: true,
      };

    case "error":
      return {
        message: "Not controlled by you. Click to take absolute control.",
        chipColor: "var(--pluto-error-z)",
      };
    case "success":
      if (status.details?.authority === clientControl.ABSOLUTE_AUTHORITY)
        return {
          message: "You have absolute control. Click to release.",
          chipColor: "var(--pluto-secondary-z)",
          buttonStyle: {
            background: "var(--pluto-secondary-z-30)",
          },
        };
      return {
        message: "You're in control. Release schematic to release control.",
        chipColor: "var(--pluto-primary-z)",
      };
    default:
      return {
        message: "Unexpected status.",
        chipColor: "var(--pluto-error-z)",
      };
  }
};

export const Chip = ({ source, sink, className, ...rest }: ChipProps): ReactElement => {
  const memoProps = useMemoDeepEqual({ source, sink });
  const [, { status }, setState] = Aether.use({
    type: control.Chip.TYPE,
    schema: control.chipStateZ,
    initialState: {
      triggered: false,
      ...memoProps,
      status: {
        key: "no_chip",
        variant: "disabled",
        message: "No chip connected.",
        time: TimeStamp.now(),
        details: {},
      },
    },
  });

  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);

  const handleToggle = useCallback(
    () => setState((state) => ({ ...state, triggered: !state.triggered })),
    [setState],
  );

  const { message, chipColor, buttonStyle, disabled } = tooltipMessage(status);

  return (
    <Button.Button
      variant="text"
      className={CSS(CSS.B("control-chip"), className)}
      disabled={disabled}
      onClick={handleToggle}
      tooltip={<Text.Text level="small">{message}</Text.Text>}
      style={buttonStyle}
      {...rest}
    >
      <Icon.Circle color={chipColor} />
    </Button.Button>
  );
};
