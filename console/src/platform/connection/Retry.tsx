// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/connection/Retry.css";

import { type connection } from "@synnaxlabs/client";
import { Button, Icon, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useCountdown } from "@/platform/connection/useCountdown";
import { useHeldChecking } from "@/platform/connection/useHeldChecking";
import { CSS } from "@/platform/css";

export interface RetryProps extends Omit<Button.ButtonProps, "onClick" | "disabled"> {}

/**
 * Runs a connection check now. Disabled while a check is in flight, and renders
 * nothing when there is no client.
 */
export const Retry = ({ children, ...rest }: RetryProps): ReactElement | null => {
  const client = Synnax.use();
  const { details } = Synnax.useConnectionStatus();
  if (client == null) return null;
  return (
    <Button.Button
      disabled={details.checking}
      onClick={() => client.connection.retryNow()}
      {...rest}
    >
      <Icon.Refresh />
      {children ?? "Retry now"}
    </Button.Button>
  );
};

interface CountdownProps {
  retry: NonNullable<connection.StatusDetails["retry"]>;
}

const Countdown = ({ retry }: CountdownProps): ReactElement => {
  const remaining = useCountdown(retry.nextAt);
  return (
    <>
      <Icon.Sync />
      {`Retrying in ${remaining}s (attempt ${retry.attempt})`}
    </>
  );
};

interface ContentProps {
  retry: connection.StatusDetails["retry"];
  checking: boolean;
}

const Content = ({ retry, checking }: ContentProps): ReactElement | null => {
  if (checking)
    return (
      <>
        <Icon.Loading />
        Checking connection
      </>
    );
  if (retry == null) return null;
  return <Countdown retry={retry} />;
};

export interface RetryScheduleProps {
  details: connection.StatusDetails;
}

/**
 * The retry schedule on one line: the check in flight, or the wait until the next
 * one. Holds its line height when there is nothing to report.
 */
export const RetrySchedule = ({
  details: { retry, checking },
}: RetryScheduleProps): ReactElement => {
  const held = useHeldChecking(checking);
  return (
    <Text.Text level="small" color={9} className={CSS.B("connection-retry")}>
      <Content retry={retry} checking={held} />
    </Text.Text>
  );
};
