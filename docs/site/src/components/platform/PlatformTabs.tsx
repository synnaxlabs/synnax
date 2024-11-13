// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Select, Text } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";

import { Tabs as Core } from "@/components/Tabs";

const TABS = [
  { key: "docker", name: "Docker", tabKey: "docker", icon: <Icon.OS.Docker /> },
  { key: "linux", name: "Linux", tabKey: "linux", icon: <Icon.OS.Linux /> },
  { key: "macos", name: "MacOS", tabKey: "macos", icon: <Icon.OS.MacOS /> },
  { key: "windows", name: "Windows", tabKey: "windows", icon: <Icon.OS.Windows /> },
];

export interface PlatformTabsProps {
  exclude?: string[];
  priority?: string[];
}

export const PlatformTabs = ({ exclude = [], priority = [], ...props }) => {
  let tabs = [
    { name: "Docker", tabKey: "docker", icon: <Icon.OS.Docker /> },
    { name: "Linux", tabKey: "linux", icon: <Icon.OS.Linux /> },
    { name: "MacOS", tabKey: "macos", icon: <Icon.OS.MacOS /> },
    { name: "Windows", tabKey: "windows", icon: <Icon.OS.Windows /> },
  ].filter((tab) => !exclude.includes(tab.tabKey));

  if (priority.length > 0)
    tabs = tabs.sort((a, b) => {
      const aIndex = priority.indexOf(a.tabKey);
      const bIndex = priority.indexOf(b.tabKey);
      // idx of -1 means not in priority list, so it should be sorted to the end
      if (bIndex === -1) return -1;
      return aIndex - bIndex;
    });
  return <Core queryParamKey="platform" tabs={tabs} {...props} />;
};

export const OSSelectButton = () => {
  const [value, onChange] = useState<string | undefined>(undefined);

  useEffect(() => {
    setInterval(() => {
      const url = new URL(window.location.href);
      const os = url.searchParams.get("platform");
      if (os != null) onChange(os);
    }, 200);
  }, []);

  const handleChange = (value) => {
    const url = new URL(window.location.href);
    url.searchParams.set("platform", value);
    window.history.pushState({}, "", url.toString());
    onChange(value);
  };

  if (value == null) return null;

  return (
    <Select.DropdownButton
      className="styled-scrollbar"
      location={{ y: "bottom" }}
      data={TABS}
      value={value}
      onChange={handleChange}
      columns={[
        {
          key: "icon",
          name: "icon",
          render: ({ entry: { name, icon } }) => (
            <Text.WithIcon level="small" startIcon={icon}>
              {name}
            </Text.WithIcon>
          ),
        },
      ]}
    >
      {({ selected: s, toggle }) => (
        <Select.BaseButton
          iconSpacing="small"
          size="medium"
          onClick={toggle}
          variant="outlined"
          startIcon={s?.icon}
          level="small"
        >
          {s?.name}
        </Select.BaseButton>
      )}
    </Select.DropdownButton>
  );
};
