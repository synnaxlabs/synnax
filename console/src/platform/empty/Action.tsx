// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Text, Triggers } from "@synnaxlabs/pluto";

export interface ActionProps
  extends
    Omit<Flex.BoxProps<"div">, "onClick">,
    Pick<Text.TextProps, "onClick" | "level"> {
  message: string;
  action?: string;
  /** Shortcut that runs the same action; rendered as a hint beside the link. */
  trigger?: Triggers.Trigger;
}

export const Action = ({
  message,
  action,
  trigger,
  onClick,
  direction,
  level,
  x,
  y = true,
  ...rest
}: ActionProps) => (
  <Flex.Box center {...rest}>
    <Text.Text
      y={y}
      x={x}
      center
      status="disabled"
      direction={direction}
      gap="tiny"
      level={level}
    >
      {message}
      {action && (
        <Text.Text onClick={onClick} variant="link" level={level}>
          {action}
        </Text.Text>
      )}
      {action && trigger && <Triggers.Text trigger={trigger} level="small" />}
    </Text.Text>
  </Flex.Box>
);
