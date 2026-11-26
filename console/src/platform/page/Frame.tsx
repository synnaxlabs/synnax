// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/page/Frame.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useResize } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { box } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import type { Group } from "@/platform/page/section";
import { Settings } from "@/platform/settings";
import { Session } from "@/session";

const useVisibleGroups = (groups: Group[]): Group[] => {
  "use no memo"; // Calls a hook per section, so the group list sets the hook order.
  const sections = groups.flatMap((group) => group.sections);
  const visibilities = sections.map((section) => section.useVisible?.() ?? true);
  const visible = new Set(sections.filter((_, i) => visibilities[i]));
  return groups
    .map((group) => ({
      ...group,
      sections: group.sections.filter((s) => visible.has(s)),
    }))
    .filter((group) => group.sections.length > 0);
};

// Below this width, the section list always shows only its icons.
const NARROW_WIDTH = 800;

export interface FrameProps {
  /** The page that the frame renders, which keeps its own selected section. */
  page: Session.Nav.Page;
  /** The name of the page, which labels its section list. */
  name: string;
  /** A module-level constant, since each section's visibility hook runs in order. */
  groups: Group[];
}

/**
 * Renders a page of sections, such as Settings: the section list on the left and the
 * selected section on the right. The window keeps the selected section of each page,
 * and shows the first section the user can use when it has none or cannot use it.
 */
export const Frame = ({ page, name, groups }: FrameProps): ReactElement => {
  const visible = useVisibleGroups(groups);
  const sections = visible.flatMap((group) => group.sections);
  const selected = Session.Nav.useSelectSection(page);
  const background = Settings.useCardBackground();
  const dispatch = Session.useDispatch();
  const [narrow, setNarrow] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const ref = useResize<HTMLDivElement>(
    (b) => {
      const width = box.width(b);
      // A hidden tab measures 0, which is not a narrow pane.
      if (width > 0) setNarrow(width < NARROW_WIDTH);
    },
    { triggers: ["x"] },
  );
  const rail = narrow || collapsed;
  const toggleLabel = collapsed ? "Expand sidebar" : "Collapse sidebar";
  const setSelected = (section: string): void => {
    dispatch(Session.Nav.selectSection({ page, section }));
  };
  const section = sections.find(({ key }) => key === selected) ?? sections[0];
  return (
    <Flex.Box x full empty ref={ref} className={CSS.B("page")}>
      <Flex.Box
        el="nav"
        aria-label={name}
        empty
        bordered
        rounded={rail ? "medium" : "huge"}
        background={background}
        className={CSS.cls(
          CSS.BE("page", "nav"),
          narrow && CSS.BEM("page", "nav", "narrow"),
          rail && CSS.BEM("page", "nav", "rail"),
        )}
      >
        {!narrow && (
          <Button.Button
            variant="text"
            size="small"
            aria-label={toggleLabel}
            tooltip={toggleLabel}
            tooltipLocation="right"
            aria-expanded={!collapsed}
            onClick={() => setCollapsed((prev) => !prev)}
            className={CSS.BE("page", "toggle")}
          >
            <Icon.Sidebar />
          </Button.Button>
        )}
        <Menu.Menu value={section?.key} onChange={setSelected}>
          {visible.map(({ key, name, sections }) => (
            <Flex.Box y key={key} gap="tiny" className={CSS.BE("page", "nav-group")}>
              <Text.Text
                level="small"
                color={9}
                className={CSS.BE("page", "nav-heading")}
              >
                {name}
              </Text.Text>
              {sections.map(({ key, name, icon }) => (
                <Menu.Item
                  key={key}
                  itemKey={key}
                  gap="medium"
                  aria-label={name}
                  tooltip={name}
                  tooltipLocation="right"
                  hideTooltip={!rail}
                  className={CSS.BE("page", "nav-item")}
                >
                  {icon}
                  <span className={CSS.BE("page", "nav-name")}>{name}</span>
                </Menu.Item>
              ))}
            </Flex.Box>
          ))}
        </Menu.Menu>
      </Flex.Box>
      <div className={CSS.BE("page", "section")}>
        {section != null && (
          <Flex.Box y className={CSS.BE("page", "content")}>
            <Text.Text level="h3" gap="medium">
              {section.icon}
              {section.name}
            </Text.Text>
            <section.Content />
          </Flex.Box>
        )}
      </div>
    </Flex.Box>
  );
};
