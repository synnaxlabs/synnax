// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Text } from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";

import { useSelect } from "@/version/selectors";

type BadgeProps<L extends Text.Level> = Optional<Text.TextProps<L>, "level">;

export const Badge = <L extends Text.Level>({
  level = "p",
  ...props
}: BadgeProps<L>): ReactElement => {
  const v = useSelect();
  return (
    <Text.Text level={level} {...props}>
      {"v" + v}
    </Text.Text>
  );
};
