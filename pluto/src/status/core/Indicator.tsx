// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { VARIANT_COLORS } from "@/status/core/colors";

export interface IndicatorProps extends Icon.IconProps {
  variant?: status.Variant;
}

export const Indicator = ({ variant, ...rest }: IndicatorProps): ReactElement =>
  variant === "loading" ? (
    <Icon.Loading {...rest} />
  ) : (
    <Icon.Circle
      color={variant != null ? VARIANT_COLORS[variant] : undefined}
      {...rest}
    />
  );
