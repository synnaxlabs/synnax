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
import { type ComponentPropsWithRef, isValidElement, type ReactElement } from "react";

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

export interface NotificationProps extends ComponentPropsWithRef<"div"> {
  status: NotificationSpec;
  silence: (key: string) => void;
  actions?: ReactElement | Button.ButtonProps[];
  /** Custom indicator glyph, tinted with the status variant color. */
  icon?: ReactElement<Icon.IconProps>;
}

export const Notification = ({
  status: stat,
  silence,
  actions,
  icon,
  className,
  children,
  ...rest
}: NotificationProps): ReactElement => {
  const { key, time, count, message, description, variant, name } = stat;
  const getCopyText = () => status.toString(stat);

  return (
    // The key remounts the card when a repeat arrives, replaying the entrance animation.
    <div
      className={CSS(CSS.B("notification"), className)}
      key={time.toString()}
      {...rest}
    >
      <Indicator variant={variant} className={CSS.BE("notification", "indicator")}>
        {icon}
      </Indicator>
      <div className={CSS.BE("notification", "text")}>
        <div className={CSS.BE("notification", "chrome")}>
          <Flex.Box
            x
            align="center"
            gap="small"
            className={CSS.BE("notification", "stamp")}
          >
            {count > 1 && <Text.Text level="small" color={9}>{`x${count}`}</Text.Text>}
            <TelemText.TimeStamp level="small" color={9} format="time">
              {time}
            </TelemText.TimeStamp>
          </Flex.Box>
          <Flex.Box x gap="tiny" className={CSS.BE("notification", "controls")}>
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
        {primitive.isNonZero(name) && (
          <Text.Text
            level="small"
            status={variant}
            weight={500}
            overflow="ellipsis"
            className={CSS.BE("notification", "name")}
          >
            {name}
          </Text.Text>
        )}
        {children ?? (
          <Text.Text className={CSS.BE("notification", "message")}>{message}</Text.Text>
        )}
        {primitive.isNonZero(description) && (
          <Text.Text
            level="small"
            color={9}
            lineClamp={8}
            className={CSS.BE("notification", "description")}
          >
            {description}
          </Text.Text>
        )}
      </div>
      {actions != null && (
        <Flex.Box
          x
          align="center"
          justify="end"
          className={CSS.BE("notification", "actions")}
        >
          {array.toArray<ReactElement | Button.ButtonProps>(actions).map((a) => (
            <Action key={a.key} action={a} />
          ))}
        </Flex.Box>
      )}
    </div>
  );
};
