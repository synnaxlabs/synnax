// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/base/Notification.css";

import { status } from "@synnaxlabs/client";
import { array, primitive } from "@synnaxlabs/x";
import { isValidElement, type ReactElement, useRef } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type NotificationSpec } from "@/status/base/Aggregator";
import { Indicator } from "@/status/base/Indicator";
import { Text as TelemText } from "@/telem/text";
import { Text } from "@/text";

interface ActionProps {
  action: ReactElement | Button.ButtonProps;
}

const Action = ({ action }: ActionProps): ReactElement =>
  isValidElement(action) ? (
    action
  ) : (
    <Button.Button {...action} key={action.key} size="tiny" />
  );

export interface NotificationProps extends Flex.BoxProps {
  status: NotificationSpec;
  silence: (key: string) => void;
  actions?: ReactElement | Button.ButtonProps[];
}

export const Notification = ({
  status: stat,
  silence,
  actions,
  className,
  children,
  ...rest
}: NotificationProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  const { key, time, count, message, description, variant, name } = stat;
  const getCopyText = () => status.toString(stat);

  return (
    <Flex.Box
      className={CSS(CSS.B("notification"), className)}
      y
      key={time.toString()}
      empty
      ref={ref}
      {...rest}
    >
      <Flex.Box x justify="between" className={CSS(CSS.BE("notification", "header"))}>
        <Text.Text
          level="small"
          status={variant}
          weight={500}
          gap="tiny"
          className={CSS(CSS.BE("notification", "name"))}
        >
          <Indicator variant={variant} />
          <Text.Text el="span" overflow="ellipsis" status={variant}>
            {primitive.isNonZero(name) && name}
          </Text.Text>
        </Text.Text>
        <div className={CSS(CSS.BE("notification", "header-end"))}>
          <Flex.Box
            x
            align="center"
            gap="small"
            className={CSS(CSS.BE("notification", "stamp"))}
          >
            {count > 1 && <Text.Text level="small" color={9}>{`x${count}`}</Text.Text>}
            <TelemText.TimeStamp level="small" color={9} format="time">
              {time}
            </TelemText.TimeStamp>
          </Flex.Box>
          <Flex.Box
            x
            className={CSS(CSS.BE("notification", "header-actions"))}
            gap="tiny"
          >
            <Button.Copy
              text={getCopyText}
              variant="text"
              size="small"
              tooltip="Copy diagnostics"
              square
              textColor={10}
            />
            <Button.Button variant="outlined" size="small" onClick={() => silence(key)}>
              <Icon.Close />
            </Button.Button>
          </Flex.Box>
        </div>
      </Flex.Box>
      <Flex.Box
        y
        align="start"
        className={CSS(CSS.BE("notification", "content"))}
        gap="small"
      >
        {children != null ? (
          children
        ) : (
          <Text.Text lineClamp={3} grow>
            {message}
          </Text.Text>
        )}
        {description != null && (
          <Text.Text level="small" color={9} lineClamp={8} grow>
            {description}
          </Text.Text>
        )}
      </Flex.Box>
      {actions != null && (
        <Flex.Box x align="center" justify="end">
          {array.toArray<ReactElement | Button.ButtonProps>(actions).map((a) => (
            <Action key={a.key} action={a} />
          ))}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
