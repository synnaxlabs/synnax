// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Status, Text, Button, List } from "@synnaxlabs/pluto";

import { CSS } from "@/css";

interface NotificationsProps extends Status.UseNotificationsReturn {}

export const Notifications = ({
  statuses,
  silence,
}: NotificationsProps): ReactElement => (
  <List.List<string, Status.Spec> data={statuses}>
    <List.Core<string, Status.Spec> className={CSS(CSS.B("notifications"))}>
      {({ entry: { key, time, message, variant } }) => (
        <Align.Space direction="x" key={time.toString()} align="center">
          <Text.DateTime level="p" format="time">
            {time}
          </Text.DateTime>
          <Status.Text variant={variant} style={{ flexGrow: 1 }}>
            {message}
          </Status.Text>
          <Button.Icon variant="text" size="small" onClick={() => silence(key)}>
            <Icon.Close />
          </Button.Icon>
        </Align.Space>
      )}
    </List.Core>
  </List.List>
);
