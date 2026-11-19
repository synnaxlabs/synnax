// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/staleness/Badge.css";

import { Badge as Base } from "@synnaxlabs/lyra/badge";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export const BADGE_CLASS = CSS.B("stale-badge");

/** Marks a symbol whose data is stale. */
export const Badge = (): ReactElement => (
  <Base.Badge status="warning" className={BADGE_CLASS} role="img" aria-label="Stale">
    <Status.Indicator variant="warning">
      <Icon.TimeOutline className={CSS.BE("stale-badge", "icon")} />
    </Status.Indicator>
  </Base.Badge>
);
