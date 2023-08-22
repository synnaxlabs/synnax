// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Text } from "@synnaxlabs/pluto";
import { Optional } from "@synnaxlabs/x";

import { useSelectVersion } from "@/version/store";

type VersionBadgeProps<L extends Text.Level> = Optional<Text.TextProps<L>, "level">;

export const VersionBadge = <L extends Text.Level>({
  level = "p",
  ...props
}: VersionBadgeProps<L>): ReactElement => {
  const v = useSelectVersion();
  return (
    <Text.Text level={level} {...props}>
      {"v" + v}
    </Text.Text>
  );
};
