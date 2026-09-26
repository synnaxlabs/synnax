// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/form.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Tabs as Base } from "@synnaxlabs/lyra/tabs";
import { type ReactElement, type ReactNode } from "react";

import { type FormProps } from "@/schematic/node/spec";

const TABS = {
  style: { name: "Style", icon: <Icon.FillColor /> },
  control: { name: "Control", icon: <Icon.Control /> },
  telemetry: { name: "Telemetry", icon: <Icon.Channel /> },
  options: { name: "Options", icon: <Icon.Menu /> },
  redline: { name: "Redline", icon: <Icon.Range /> },
  fill: { name: "Fill", icon: <Icon.Tank /> },
};

export type TabKey = keyof typeof TABS;

export interface TabsProps extends Pick<FormProps, "actions"> {
  /** The rail's tabs, in order. The first is selected initially. */
  tabs: TabKey[];
  /** A `Tabs.Content` per tab key. */
  children: ReactNode;
}

/** Tabs lays a symbol form's sub-tabs in a rail beside the content. */
export const Tabs = ({ tabs, actions, children }: TabsProps): ReactElement => (
  <Base.Frame initialValue={tabs[0]} x className={CSS.B("symbol-form-tabs")}>
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
