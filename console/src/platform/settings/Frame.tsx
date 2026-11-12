// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/settings/Frame.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { type Group, useVisibleGroups } from "@/platform/settings/section";

export interface FrameProps {
  /** A module-level constant, since each section's visibility hook runs in order. */
  groups: Group[];
}

/**
 * Renders Settings: the section list on the left and the selected section's page on
 * the right. The first section the user can use starts selected.
 */
export const Frame = ({ groups }: FrameProps): ReactElement => {
  const visible = useVisibleGroups(groups);
  const sections = visible.flatMap((group) => group.sections);
  const [selected, setSelected] = useState(sections[0]?.key);
  const section = sections.find(({ key }) => key === selected) ?? sections[0];
  return (
    <Flex.Box x full empty className={CSS.B("settings")}>
      <nav aria-label="Settings" className={CSS.BE("settings", "nav")}>
        <Menu.Menu value={section?.key} onChange={setSelected}>
          {visible.map(({ key, name, sections }) => (
            <Flex.Box
              y
              key={key}
              gap="tiny"
              className={CSS.BE("settings", "nav-group")}
            >
              <Text.Text
                level="small"
                color={9}
                className={CSS.BE("settings", "nav-heading")}
              >
                {name}
              </Text.Text>
              {sections.map(({ key, name, icon }) => (
                <Menu.Item key={key} itemKey={key} gap="medium">
                  {icon}
                  {name}
                </Menu.Item>
              ))}
            </Flex.Box>
          ))}
        </Menu.Menu>
      </nav>
      <div className={CSS.BE("settings", "page")}>
        {section != null && <section.Content />}
      </div>
    </Flex.Box>
  );
};
