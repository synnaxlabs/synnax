// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";

import { Space, Text, TypographyLevel } from "@/core";

export interface ValueProps {
  value: number;
  level?: TypographyLevel;
  label?: string;
  variant?: "primary" | "error";
  color?: string;
}

export const Statistic = ({
  value,
  level = "h4",
  variant = "primary",
  label,
}: ValueProps): JSX.Element => {
  return (
    <Space empty align="center" justify="center">
      <Text
        className={clsx(
          "pluto-value__text",
          variant.length > 0 && `pluto-value__text--${variant}`
        )}
        level={level}
      >
        {value}
      </Text>
      {label != null && (
        <Text className="pluto-value__label" level="small">
          {label}
        </Text>
      )}
    </Space>
  );
};
