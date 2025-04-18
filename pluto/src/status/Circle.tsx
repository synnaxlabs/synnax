// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, type IconProps } from "@synnaxlabs/media";
import { status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { VARIANT_COLORS } from "@/status/colors";

export interface CircleProps extends IconProps {
  variant?: status.Variant;
  disabled?: boolean;
  loading?: boolean;
}

export const Circle = ({
  variant = status.INFO_VARIANT,
  disabled = false,
  loading = false,
  ...rest
}: CircleProps): ReactElement =>
  loading ? (
    <Icon.Loading {...rest} />
  ) : (
    <Icon.Circle
      color={disabled ? "var(--pluto-gray-l8)" : VARIANT_COLORS[variant]}
      {...rest}
    />
  );
