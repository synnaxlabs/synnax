// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "@synnaxlabs/pluto";
import type { TextProps } from "@synnaxlabs/pluto";
import { Optional } from "@synnaxlabs/x";

import { useSelectVersion } from "../store";

type VersionBadgeProps = Optional<TextProps, "level">;

export const VersionBadge = ({
  level = "p",
  ...props
}: VersionBadgeProps): JSX.Element => {
  const v = useSelectVersion();
  return (
    <Text level={level} {...props}>
      {"v" + v}
    </Text>
  );
};
