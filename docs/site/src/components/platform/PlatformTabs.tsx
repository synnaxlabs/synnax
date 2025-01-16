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
import { runtime } from "@synnaxlabs/x";
import { useEffect, useLayoutEffect, useState } from "react";

import { Tabs as Core } from "@/components/Tabs";

const TABS = [
  { key: "docker", name: "Docker", tabKey: "docker", icon: <Icon.OS.Docker /> },
  { key: "linux", name: "Linux", tabKey: "linux", icon: <Icon.OS.Linux /> },
  { key: "macos", name: "macOS", tabKey: "macos", icon: <Icon.OS.MacOS /> },
  { key: "windows", name: "Windows", tabKey: "windows", icon: <Icon.OS.Windows /> },
];

export interface PlatformTabsProps {
  exclude?: string[];
  priority?: string[];
}

const getOSFromURL = (detect: boolean): runtime.OS | null => {
  const url = new URL(window.location.href);
  const p = url.searchParams.get("platform");
  const os = runtime.osZ.safeParse(p);
  if (!os.success) return detect ? runtime.getOS() : null;
  return os.data;
};

const setOSInURL = (os: runtime.OS) => {
  const url = new URL(window.location.href);
  url.searchParams.set("platform", os.toLowerCase());
  window.history.pushState({}, "", url.toString());
};

export const PlatformTabs = ({ exclude = [], priority = [], ...props }) => {
  let tabs = [
    { name: "Docker", tabKey: "docker", icon: <Icon.OS.Docker /> },
    { name: "Linux", tabKey: "linux", icon: <Icon.OS.Linux /> },
    { name: "macOS", tabKey: "macos", icon: <Icon.OS.MacOS /> },
    { name: "Windows", tabKey: "windows", icon: <Icon.OS.Windows /> },
  ].filter((tab) => !exclude.includes(tab.tabKey));

  useLayoutEffect(() => {
    const os = getOSFromURL(true);
    if (os != null) setOSInURL(os);
  }, []);

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
    const i = setInterval(() => {
      const os = getOSFromURL(false);
      if (os != null) onChange(os.toLowerCase());
    }, 200);
    return () => clearInterval(i);
  }, []);

  const handleChange = (value: runtime.OS) => {
    setOSInURL(value);
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
