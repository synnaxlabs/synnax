// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/Notification.css";

import { array } from "@synnaxlabs/x";
import { isValidElement, type ReactElement, useRef } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type NotificationSpec } from "@/status/Aggregator";
import { Indicator } from "@/status/Indicator";
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
  status: { key, time, count, message, description, variant },
  silence,
  actions,
  className,
  children,
  ...rest
}: NotificationProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);

  return (
    <Flex.Box
      className={CSS(CSS.B("notification"), className)}
      y
      key={time.toString()}
      empty
      ref={ref}
      {...rest}
    >
      <Flex.Box x justify="between" grow style={{ width: "100%" }}>
        <Flex.Box x align="center" gap="small">
          <Indicator style={{ height: "2.25rem", width: "2.5rem" }} variant={variant} />
          <Text.Text level="small">{`x${count}`}</Text.Text>
          <Text.DateTime
            className={CSS(CSS.BE("notification", "time"))}
            level="small"
            format="time"
          >
            {time}
          </Text.DateTime>
        </Flex.Box>
        <Button.Button
          className={CSS(CSS.BE("notification", "silence"))}
          variant="outlined"
          size="small"
          onClick={() => silence(key)}
        >
          <Icon.Close />
        </Button.Button>
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
          <Text.Text
            className={CSS(CSS.BE("notification", "message"))}
            style={{ flexGrow: 1 }}
          >
            {message}
          </Text.Text>
        )}
        {description != null && (
          <Text.Text
            className={CSS(CSS.BE("notification", "description"))}
            level="small"
            style={{ flexGrow: 1 }}
          >
            {description}
          </Text.Text>
        )}
      </Flex.Box>
      {actions != null && (
        <Flex.Box
          x
          align="center"
          justify="end"
          className={CSS(CSS.BE("notification", "actions"))}
        >
          {array.toArray<ReactElement | Button.ButtonProps>(actions).map((a) => (
            <Action key={a.key} action={a} />
          ))}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
