// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Text } from "@synnaxlabs/pluto";

export interface EmptyActionProps
  extends Omit<Flex.BoxProps<"div">, "onClick">, Pick<Text.TextProps, "onClick"> {
  message: string;
  action?: string;
}

export const EmptyAction = ({
  message,
  action,
  onClick,
  direction,
  x,
  y = true,
  ...rest
}: EmptyActionProps) => (
  <Flex.Box center {...rest}>
    <Text.Text y={y} x={x} center status="disabled" direction={direction} gap="tiny">
      {message}
      {action && (
        <Text.Text onClick={onClick} variant="link">
          {action}
        </Text.Text>
      )}
    </Text.Text>
  </Flex.Box>
);
