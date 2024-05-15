// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useMemo, isValidElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Status, Text } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { List } from "@synnaxlabs/pluto/list";
import { TimeSpan, toArray } from "@synnaxlabs/x";

import { CSS } from "@/css";

import "@/notifications/Notifications.css";
import { notificationAdapter } from "@/hardware/device/useListenForChanges";

interface NotificationsProps {
  adapters?: NotificationAdapter[];
}

interface SugaredNotification extends Status.Notification {
  actions?: ReactElement | Button.ButtonProps[];
}

export type NotificationAdapter = (
  status: Status.Notification,
) => null | SugaredNotification;

export const Notifications = ({ adapters }: NotificationsProps): ReactElement => {
  adapters = [notificationAdapter];
  const { statuses, silence } = Status.useNotifications({
    expiration: TimeSpan.seconds(5000),
  });

  return (
    <List.List<string, Status.Notification> data={statuses}>
      <List.Core<string, Status.Notification>
        className={CSS(CSS.B("notifications"))}
        size="medium"
      >
        {({ entry }) => (
          <Notification
            key={entry.key}
            status={entry}
            adapters={adapters}
            silence={silence}
          />
        )}
      </List.Core>
    </List.List>
  );
};

interface NotificationProps {
  status: Status.Notification;
  adapters?: NotificationAdapter[];
  silence: (key: string) => void;
}

const Notification = ({
  status,
  adapters,
  silence,
}: NotificationProps): ReactElement => {
  const adapted: Status.Notification | SugaredNotification = useMemo(() => {
    if (adapters == null || adapters.length === 0) return status;
    for (const adapter of adapters) {
      const result = adapter(status);
      if (result != null) return result;
    }
    return status;
  }, [status.key, adapters]);
  const { key, time, message, description, variant, count } = adapted;
  return (
    <Align.Space
      className={CSS(CSS.B("notification"))}
      direction="y"
      key={time.toString()}
      empty
    >
      <Align.Space direction="x" justify="spaceBetween" grow style={{ width: "100%" }}>
        <Align.Space direction="x" align="center" size="small">
          <Status.Circle
            style={{ height: "2.25rem", width: "2.5rem" }}
            variant={variant}
          />
          <Text.Text level="small" shade={7}>
            {`x${count}`}
          </Text.Text>
          <Text.DateTime
            className={CSS(CSS.BE("notification", "time"))}
            level="small"
            format="time"
          >
            {time}
          </Text.DateTime>
        </Align.Space>
        <Button.Icon
          className={CSS(CSS.BE("notification", "silence"))}
          variant="text"
          size="small"
          onClick={() => silence(key)}
        >
          <Icon.Close />
        </Button.Icon>
      </Align.Space>
      <Align.Space
        direction="y"
        align="start"
        className={CSS(CSS.BE("notification", "content"))}
        size="small"
      >
        <Text.Text
          className={CSS(CSS.BE("notification", "message"))}
          level="p"
          style={{ flexGrow: 1 }}
        >
          {message}
        </Text.Text>
        {description != null && (
          <Text.Text
            className={CSS(CSS.BE("notification", "description"))}
            level="small"
            style={{ flexGrow: 1 }}
          >
            This is the description of my action and this is the even more multi-line
            than the message.
          </Text.Text>
        )}
      </Align.Space>
      {"actions" in adapted && adapted.actions != null && (
        <Align.Space
          direction="x"
          align="center"
          justify="end"
          className={CSS(CSS.BE("notification", "actions"))}
        >
          {toArray<ReactElement | Button.ButtonProps>(adapted.actions).map(
            (action, i) => (
              <Action key={i} action={action} />
            ),
          )}
        </Align.Space>
      )}
    </Align.Space>
  );
};

interface ActionProps {
  action: ReactElement | Button.ButtonProps;
}

const Action = ({ action }: ActionProps): ReactElement => {
  if (!isValidElement(action)) {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    const props = action as Button.ButtonProps;
    return <Button.Button {...props} />;
  }
  return action;
};
