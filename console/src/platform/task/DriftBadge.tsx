// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";
import { useDrifted } from "@/platform/task/useDrifted";

/** Warns that the running instance no longer matches the stored config. */
export const DriftBadge = () => {
  const drifted = useDrifted();
  if (!drifted) return null;
  return (
    <Text.Text
      className={CSS.B("task-drift-badge")}
      level="small"
      status="warning"
      weight={450}
    >
      <Icon.Warning />
      Configuration changed since deploy
    </Text.Text>
  );
};
