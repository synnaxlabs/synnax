// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { context } from "@synnaxlabs/lyra/context";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Portal } from "@synnaxlabs/lyra/portal";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { BROWSE_DURATION } from "@/feature/mqtt/device/browse";
import { CSS } from "@/platform/css";

export const LISTENING = `Listening for ${BROWSE_DURATION.seconds} seconds`;

const [ActiveContext, useActiveContext] = context.create<string>({
  displayName: "Picker.ActiveContext",
  providerName: "Picker.Root",
});

const [FrameContext, useFrameContext] = context.create<string>({
  displayName: "Picker.FrameContext",
  providerName: "Picker.Frame",
});

export interface RootProps extends PropsWithChildren {
  /** The key of the frame that shows. */
  value: string;
}

/** Shows one {@link Frame} at a time and puts its {@link Status} in the {@link Bar}. */
export const Root = ({ value, children }: RootProps): ReactElement => (
  <Portal.Context>
    <ActiveContext value={value}>{children}</ActiveContext>
  </Portal.Context>
);

/** The bar above the frames: its children, then the status of the frame that shows. */
export const Bar = ({ children }: PropsWithChildren): ReactElement => (
  <Nav.Bar
    location="top"
    size="8rem"
    bordered
    className={CSS.BE("mqtt-discover", "bar")}
  >
    <Nav.Bar.Start>{children}</Nav.Bar.Start>
    <Nav.Bar.End align="center">
      <Portal.Out itemKey={useActiveContext("Picker.Bar")} />
    </Nav.Bar.End>
  </Nav.Bar>
);

export interface FrameProps extends PropsWithChildren {
  itemKey: string;
}

/**
 * The body of one tab. It mounts its children the first time it shows and keeps them
 * mounted while hidden, so a picker listens only once opened and keeps what it shows.
 */
export const Frame = ({ itemKey, children }: FrameProps): ReactElement | null => {
  const active = useActiveContext("Picker.Frame") === itemKey;
  const [opened, setOpened] = useState(active);
  if (active && !opened) setOpened(true);
  if (!opened) return null;
  return (
    <FrameContext value={itemKey}>
      <Flex.Box
        x
        grow
        empty
        className={CSS.cls(
          CSS.BE("mqtt-discover", "frame"),
          !active && CSS.M("hidden"),
        )}
      >
        {children}
      </Flex.Box>
    </FrameContext>
  );
};

export interface StatusProps {
  status: string;
  loading: boolean;
  onListen: () => void;
}

/** The state of the listen of its {@link Frame}, shown in the {@link Bar}. */
export const Status = ({ status, loading, onListen }: StatusProps): ReactElement => (
  <Portal.In itemKey={useFrameContext("Picker.Status")}>
    <Flex.Box x align="center" gap="small">
      <Text.Text level="small" color={9}>
        {status}
      </Text.Text>
      <Button.Button
        variant="text"
        size="small"
        tooltip="Listen again"
        aria-label="Listen again"
        disabled={loading}
        onClick={onListen}
      >
        <Icon.Refresh />
      </Button.Button>
      {loading && (
        <Progress.Progress
          thin
          duration={BROWSE_DURATION}
          className={CSS.BE("mqtt-discover", "progress")}
        />
      )}
    </Flex.Box>
  </Portal.In>
);

/** The column that lists what the broker publishes. */
export const Side = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y empty className={CSS.BE("mqtt-discover", "side")}>
    {children}
  </Flex.Box>
);

/** The area that shows the values of what the side selects. */
export const Pane = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y grow empty className={CSS.BE("mqtt-discover", "pane")}>
    {children}
  </Flex.Box>
);

/** A checkbox that holds the place of one still loading. */
export const CheckSkeleton = (): ReactElement => (
  <Input.Checkbox size="small" value={false} onChange={() => {}} disabled />
);
