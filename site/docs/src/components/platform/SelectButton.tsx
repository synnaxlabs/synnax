// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { useEffect, useState } from "react";

import {
  getFromURL,
  type Platform,
  PLATFORMS,
  setInURL,
} from "@/components/platform/Platform";

const indexMap = new Map<Platform, number>();
PLATFORMS.forEach((p, i) => indexMap.set(p.key, i));

export interface SelectButtonProps {
  platforms: Platform[];
}

export const SelectButton = ({ platforms }: SelectButtonProps) => {
  const [platform, setPlatform] = useState<Platform>(platforms[0]);

  // Map the platforms so the order of the platforms is consistent between the props and
  // the options of the select button.
  const options = platforms.map((p) => PLATFORMS[indexMap.get(p) as number]);

  useEffect(() => {
    const updateFromURL = () => {
      const p = getFromURL(false);
      if (p) setPlatform(p);
    };
    updateFromURL();
    window.addEventListener("popstate", updateFromURL);
    window.addEventListener("urlchange", updateFromURL);
    return () => {
      window.removeEventListener("popstate", updateFromURL);
      window.removeEventListener("urlchange", updateFromURL);
    };
  }, []);

  const handleChange = (p: Platform) => {
    setInURL(p);
    setPlatform(p);
  };

  return (
    <Select.Simple<Platform>
      className="styled-scrollbar"
      location="bottom"
      resourceName="platform"
      value={platform}
      allowNone={false}
      onChange={handleChange}
    >
      {options.map(({ key, name, icon }) => (
        <Select.Item key={key} itemKey={key}>
          {icon}
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};
