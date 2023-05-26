// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Optional } from "@synnaxlabs/x";
import clsx from "clsx";

import { useHeaderContext } from "@/core/Header/Header";
import "@/core/Header/Header.css";
import { TextWithIconProps, Text } from "@/core/Typography";

export interface HeaderTitleProps
  extends Optional<Omit<TextWithIconProps, "divided">, "level"> {}

export const HeaderTitle = ({
  className,
  level: propsLevel,
  ...props
}: HeaderTitleProps): JSX.Element => {
  const { level, divided } = useHeaderContext();
  return (
    <Text.WithIcon
      className={clsx("pluto-header__text", className)}
      level={propsLevel ?? level}
      divided={divided}
      {...props}
    />
  );
};
