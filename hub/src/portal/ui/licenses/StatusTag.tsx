// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Tag } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type LicenseStatus } from "@/portal/ui/format";

const VARIANTS: Record<LicenseStatus, "success" | "warning" | "error"> = {
  active: "success",
  expired: "warning",
  revoked: "error",
};

const LABELS: Record<LicenseStatus, string> = {
  active: "Active",
  expired: "Expired",
  revoked: "Revoked",
};

export const StatusTag = ({ status }: { status: LicenseStatus }): ReactElement => (
  <Tag.Tag icon={<Status.Indicator variant={VARIANTS[status]} />} size="small">
    {LABELS[status]}
  </Tag.Tag>
);
