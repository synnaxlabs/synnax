// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/properties/Tabs.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Tabs as Base } from "@synnaxlabs/lyra/tabs";
import { type ReactElement, type ReactNode } from "react";
import { z } from "zod";

export const tabKeyZ = z.enum([
  "style",
  "control",
  "telemetry",
  "options",
  "redline",
  "fill",
]);
export type TabKey = z.infer<typeof tabKeyZ>;

const TABS: Record<TabKey, { name: string; icon: ReactElement }> = {
  style: { name: "Style", icon: <Icon.FillColor /> },
  control: { name: "Control", icon: <Icon.Control /> },
  telemetry: { name: "Telemetry", icon: <Icon.Channel /> },
  options: { name: "Options", icon: <Icon.Menu /> },
  redline: { name: "Redline", icon: <Icon.Redline /> },
  fill: { name: "Fill", icon: <Icon.Tank /> },
};

/** The selected tab of a {@link Tabs}, owned by the caller. */
export interface SelectionProps {
  /** The selected tab. When absent or not in the rail, the first tab shows. */
  tab?: TabKey;
  onTabChange?: (tab: TabKey) => void;
}

export interface TabsProps extends SelectionProps {
  /** Renders at the foot of the rail. */
  actions?: ReactNode;
  /** The rail's tabs, in order. */
  tabs: TabKey[];
  /** A `Tabs.Content` per tab key. */
  children: ReactNode;
}

/**
 * Tabs lays a properties form's sub-tabs in a rail beside the content. With
 * `onTabChange` the caller owns the selection; without it the rail owns it.
 */
export const Tabs = ({
  tabs,
  tab,
  onTabChange,
  actions,
  children,
}: TabsProps): ReactElement => {
  const selected = tabs.find((key) => key === tab) ?? tabs[0];
  return (
    <Base.Frame
      initialValue={selected}
      value={onTabChange == null ? undefined : selected}
      onChange={(key: string) => onTabChange?.(tabKeyZ.parse(key))}
      x
      className={CSS.B("properties-tabs")}
    >
      <Base.Selector y>
        {tabs.map((key) => (
          <Base.Tab key={key} itemKey={key}>
            {TABS[key].icon}
            {TABS[key].name}
          </Base.Tab>
        ))}
        {actions != null && (
          <>
            <Flex.Box grow />
            <Flex.Box x align="center" empty>
              {actions}
            </Flex.Box>
          </>
        )}
      </Base.Selector>
      {children}
    </Base.Frame>
  );
};
