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

import { type Status as ThreadState } from "@/server/support/tracker";

export const ThreadStatus = ({ status }: { status: ThreadState }): ReactElement => (
  <Tag.Tag
    icon={<Status.Indicator variant={status === "done" ? "success" : "info"} />}
    size="small"
  >
    {status === "done" ? "Resolved" : "Open"}
  </Tag.Tag>
);
