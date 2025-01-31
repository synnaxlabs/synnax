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
import { type ReactElement, useEffect, useLayoutEffect, useState } from "react";
import { z } from "zod";

import { Tabs } from "@/components/Tabs";

const platformTypeZ = z.enum(["docker", "linux", "macos", "windows"]);
type PlatformType = z.infer<typeof platformTypeZ>;

interface PlatformTab {
  key: PlatformType;
  name: string;
  tabKey: PlatformType;
  icon: ReactElement;
}

const TABS: PlatformTab[] = [
  { key: "docker", name: "Docker", tabKey: "docker", icon: <Icon.Logo.Docker /> },
  { key: "linux", name: "Linux", tabKey: "linux", icon: <Icon.Logo.Linux /> },
  { key: "macos", name: "macOS", tabKey: "macos", icon: <Icon.Logo.Apple /> },
  { key: "windows", name: "Windows", tabKey: "windows", icon: <Icon.Logo.Windows /> },
];

const getTypeFromURL = (detect: boolean): PlatformType | null => {
  const url = new URL(window.location.href);
  const p = url.searchParams.get("platform");
  return (
    platformTypeZ.safeParse(p).data ??
    (detect ? (platformTypeZ.safeParse(runtime.getOS()).data ?? null) : null)
  );
};

const setTypeInURL = (platformType: PlatformType) => {
  const url = new URL(window.location.href);
  url.searchParams.set("platform", platformType);
  window.history.pushState({}, "", url.toString());
};

export const PlatformTabs = ({
  exclude = new Set<PlatformType>(),
  priority = Array<PlatformType>(),
  ...props
}) => {
  let tabs = TABS.filter((tab) => !exclude.has(tab.key));
  useLayoutEffect(() => {
    const os = getTypeFromURL(true);
    if (os != null) setTypeInURL(os);
  }, []);

  if (priority.length > 0)
    tabs = tabs.sort((a, b) => {
      const aIndex = priority.indexOf(a.tabKey);
      const bIndex = priority.indexOf(b.tabKey);
      // idx of -1 means not in priority list, so it should be sorted to the end
      if (bIndex === -1) return -1;
      return aIndex - bIndex;
    });
  return <Tabs queryParamKey="platform" tabs={tabs} {...props} />;
};

export const OSSelectButton = () => {
  const [value, setValue] = useState<string | undefined>(undefined);

  useEffect(() => {
    const i = setInterval(() => {
      const os = getTypeFromURL(false);
      if (os != null) setValue(os.toLowerCase());
    }, 200);
    return () => clearInterval(i);
  }, []);

  const handleChange = (value: PlatformType) => {
    setTypeInURL(value);
    setValue(value);
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
