// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, type ReactElement } from "react";

import { TimeStamp } from "@synnaxlabs/x";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { chip } from "@/control/chip/aether";
import { useMemoDeepEqualProps } from "@/memo";
import { Status } from "@/status";
import { Text } from "@/text";

export interface ChipProps
  extends Omit<z.input<typeof chip.chipStateZ>, "trigger" | "status">,
    Omit<Button.IconProps, "onClick" | "children"> {}

export const Chip = Aether.wrap<ChipProps>(
  chip.Chip.TYPE,
  ({ aetherKey, source, sink, ...props }): ReactElement => {
    const memoProps = useMemoDeepEqualProps({ source, sink });

    const [, { status }, setState] = Aether.use({
      aetherKey,
      type: chip.Chip.TYPE,
      initialState: {
        ...memoProps,
        trigger: 0,
        status: {
          key: "no_chip",
          variant: "warning",
          message: "No chip connected.",
          time: TimeStamp.now(),
        },
      },
      schema: chip.chipStateZ,
    });

    useEffect(() => {
      setState((p) => ({ ...p, ...memoProps }));
    }, [memoProps, setState]);

    const handleClick = (): void =>
      setState((state) => ({
        ...state,
        trigger: state.trigger + 1,
      }));

    return (
      <Button.Icon
        variant="text"
        onClick={handleClick}
        tooltip={<Text.Text level="small">{status.message}</Text.Text>}
        {...props}
      >
        <Status.Circle variant={status.variant} />
      </Button.Icon>
    );
  },
);
