// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { type CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Skeleton } from "@synnaxlabs/lyra/skeleton";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, type ReactNode } from "react";

import { BROWSE_DURATION } from "@/feature/mqtt/device/browse";
import { CSS } from "@/platform/css";

export const plural = (n: number, noun: string): string =>
  `${n} ${n === 1 ? noun : `${noun}s`}`;

export const LISTENING = `Listening for ${BROWSE_DURATION.seconds} seconds`;

/** The height of an item in a side tree. Its skeleton matches it. */
export const ITEM_HEIGHT = 33;

/** A checkbox that holds the place of one still loading. */
export const CheckSkeleton = () => (
  <Input.Checkbox size="small" value={false} onChange={() => {}} disabled />
);

/** A skeleton item in a side tree: its depth and the width of its name. */
export type SkeletonItem = [depth: number, width: string];

const indent = (depth: number): PCSS.VarProperties => ({ "--console-depth": depth });

export interface TreeSkeletonProps {
  items: SkeletonItem[];
}

export const TreeSkeleton = ({ items }: TreeSkeletonProps) => (
  <div aria-hidden className={CSS.BE("mqtt-discover", "skeleton")}>
    {items.map(([depth, width], i) => (
      <Flex.Box
        key={i}
        x
        align="center"
        gap="small"
        style={indent(depth)}
        className={CSS.BE("mqtt-discover", "skeleton-item")}
      >
        <CheckSkeleton />
        <Text.Text level="small">
          <Skeleton.Skeleton style={{ width }} />
        </Text.Text>
      </Flex.Box>
    ))}
  </div>
);

export interface PickerProps {
  tabs: ReactElement;
  status: string;
  loading: boolean;
  hidden: boolean;
  onRefresh: () => void;
  side: ReactNode;
  pane: ReactNode;
}

/** One tab of the dialog: a bar with the listen state, a side tree, and a pane. */
export const Picker = ({
  tabs,
  status,
  loading,
  hidden,
  onRefresh,
  side,
  pane,
}: PickerProps) => (
  <Flex.Box
    y
    grow
    empty
    className={CSS.cls(CSS.BE("mqtt-discover", "picker"), hidden && CSS.M("hidden"))}
  >
    <Nav.Bar
      location="top"
      size="8rem"
      bordered
      className={CSS.BE("mqtt-discover", "bar")}
    >
      <Nav.Bar.Start>{tabs}</Nav.Bar.Start>
      <Nav.Bar.End x align="center" gap="small">
        <Text.Text level="small" className={CSS.BE("mqtt-discover", "muted")}>
          {status}
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          tooltip="Listen again"
          aria-label="Listen again"
          disabled={loading}
          onClick={onRefresh}
        >
          <Icon.Refresh />
        </Button.Button>
      </Nav.Bar.End>
      {loading && (
        <Progress.Progress
          thin
          duration={BROWSE_DURATION}
          className={CSS.BE("mqtt-discover", "progress")}
        />
      )}
    </Nav.Bar>
    <Flex.Box x grow empty className={CSS.BE("mqtt-discover", "body")}>
      <Flex.Box y empty className={CSS.BE("mqtt-discover", "side")}>
        {side}
      </Flex.Box>
      <Flex.Box y grow empty className={CSS.BE("mqtt-discover", "pane")}>
        {pane}
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
