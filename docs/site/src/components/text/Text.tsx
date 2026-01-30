// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Text } from "@synnaxlabs/pluto";
import { type FC } from "react";

interface TextFactoryProps {
  level: Text.Level;
  includeAnchor?: boolean;
}

export interface TextProps extends Omit<Text.TextProps, "level"> {}

export const textFactory = ({
  level,
  includeAnchor = false,
}: TextFactoryProps): FC<TextProps> => {
  const Component = ({ children, id, ...p }: TextProps) => (
    <Text.Text id={id} level={level} {...p}>
      {children}
      {includeAnchor && (
        <a href={`#${id}`} className="heading-anchor">
          <Icon.Link />
        </a>
      )}
    </Text.Text>
  );
  Component.displayName = `Text.${level}`;
  return Component;
};
