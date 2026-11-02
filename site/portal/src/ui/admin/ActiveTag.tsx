// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { type ReactElement } from "react";

/** ActiveTag marks a daily active user: one active on the UTC day of today. */
export const ActiveTag = (): ReactElement => (
  <Tag.Tag icon={<Status.Indicator variant="success" />} size="small">
    Active today
  </Tag.Tag>
);
