// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Text } from "@synnaxlabs/pluto/text";

export interface StepTextProps extends Text.TextProps {
  step: string | number;
}

export const StepText = ({ step, children, ...props }: StepTextProps) => (
  <Text.Text {...props}>
    <span
      style={{
        color: "var(--pluto-gray-l7)",
        display: "inline-flex",
        alignItems: "center",
      }}
    >
      Step {step} <Icon.Arrow.Right style={{ margin: "0 1rem" }} />
    </span>
    {children}
  </Text.Text>
);
