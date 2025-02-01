// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select, Text } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";

import {
  getPlatformFromURL,
  type Platform,
  PLATFORMS,
  setPlatformInURL,
} from "@/components/platform/platform";

export interface SelectPlatformButtonProps {
  platforms: Platform[];
}

const PlatformIndexMap = new Map<Platform, number>();
PLATFORMS.forEach((p, i) => PlatformIndexMap.set(p.key, i));

export const SelectPlatformButton = ({ platforms }: SelectPlatformButtonProps) => {
  const [platform, setPlatform] = useState<Platform>(platforms[0]);

  const data = platforms.map((p) => PLATFORMS[PlatformIndexMap.get(p) as number]);

  useEffect(() => {
    const i = setInterval(() => {
      const p = getPlatformFromURL(false);
      if (p) setPlatform(p);
    }, 200);
    return () => clearInterval(i);
  }, []);

  const handleChange = (p: Platform) => {
    setPlatformInURL(p);
    setPlatform(p);
  };

  return (
    <Select.DropdownButton
      className="styled-scrollbar"
      location="bottom"
      data={data}
      value={platform}
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
      {(p) => (
        <Select.BaseButton
          {...p}
          iconSpacing="small"
          size="medium"
          variant="outlined"
          startIcon={p.selected?.icon}
          level="small"
        >
          {p.selected?.name}
        </Select.BaseButton>
      )}
    </Select.DropdownButton>
  );
};
