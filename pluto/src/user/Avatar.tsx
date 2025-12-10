// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/user/Avatar.css";

import { useMemo } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type Flex } from "@/flex";

const stringToHash = (str: string): number => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
    hash |= 0;
  }
  return Math.abs(hash);
};

const hashToHSL = (hash: number, offset: number = 0): string => {
  const hue = (hash + offset) % 360;
  const saturation = 60 + (hash % 30);
  const lightness = 50 + (hash % 10);
  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
};

export const avatar = (username: string): string => {
  const baseHash = stringToHash(username);
  const color1 = hashToHSL(baseHash, 0);
  const color2 = hashToHSL(baseHash, 120);
  return `linear-gradient(135deg, ${color1}, ${color2})`;
};

export interface AvatarProps extends Flex.BoxProps {
  username: string;
}

export const Avatar = ({
  username,
  className,
  square,
  style,
  size,
  ...rest
}: AvatarProps) => {
  const oStyle = useMemo(
    () => ({ background: avatar(username), ...style }),
    [username, style],
  );
  return (
    <Button.Button
      preventClick
      variant="text"
      {...rest}
      className={CSS(className, CSS.B("avatar"))}
      square
      style={oStyle}
    />
  );
};
