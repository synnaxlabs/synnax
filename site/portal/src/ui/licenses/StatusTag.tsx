// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status as LyraStatus } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import type { ReactElement } from "react";

import type { Status } from "@/server/license/status";
import { STATUS_LABELS } from "@/ui/format";

const VARIANTS: Record<Status, "success" | "warning" | "error"> = {
  active: "success",
  expiring: "warning",
  expired: "error",
  revoked: "error",
};

export const StatusTag = ({ status }: { status: Status }): ReactElement => (
  <Tag.Tag icon={<LyraStatus.Indicator variant={VARIANTS[status]} />} size="small">
    {STATUS_LABELS[status]}
  </Tag.Tag>
);
