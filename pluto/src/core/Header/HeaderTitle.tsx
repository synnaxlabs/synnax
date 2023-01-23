// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";

import { useHeaderContext } from "./Header";

import { TextWithIconProps, Text } from "@/core/Typography";

import "./Header.css";

export interface HeaderTitleProps
  extends Omit<TextWithIconProps, "level" | "divided"> {}

export const HeaderTitle = ({ className, ...props }: HeaderTitleProps): JSX.Element => {
  const { level, divided } = useHeaderContext();
  return (
    <Text.WithIcon
      className={clsx("pluto-header__text", className)}
      level={level}
      divided={divided}
      {...props}
    />
  );
};
