// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Optional } from "@synnaxlabs/x";

import { CSS } from "@/core/css";

import "@/core/Header/Header.css";

import { useHeaderContext } from "@/core/std/Header/Header";
import { TextWithIconProps, Text } from "@/core/std/Typography";

export interface HeaderTitleProps
  extends Optional<Omit<TextWithIconProps, "divided">, "level"> {}

export const HeaderTitle = ({
  className,
  level: propsLevel,
  ...props
}: HeaderTitleProps): ReactElement => {
  const { level, divided } = useHeaderContext();
  return (
    <Text.WithIcon
      className={CSS(CSS.BE("header", "text"), className)}
      level={propsLevel ?? level}
      divided={divided}
      {...props}
    />
  );
};
