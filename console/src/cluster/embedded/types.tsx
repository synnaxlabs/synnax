// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type Status } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type EmbeddedState } from "@/cluster/migrations";
import { type Layout } from "@/layout";

export const EMBEDDED_CONTROLS_LAYOUT_TYPE = "embeddedControls";

export const controlsLayout: Layout.State = {
  key: "embedded-controls",
  type: "embeddedControls",
  name: "Embedded Controls",
  icon: "Cluster",
  windowKey: "embedded",
  location: "modal",
  window: {
    navTop: true,
    size: {
      width: 800,
      height: 500,
    },
  },
};

export const STATUS_MAP: Record<EmbeddedState["status"], Status.Variant> = {
  running: "success",
  stopped: "error",
  stopping: "warning",
  starting: "warning",
  killed: "error",
};

export const ICON_MAP: Record<EmbeddedState["status"], ReactElement> = {
  running: <Icon.Pause />,
  killed: <Icon.Stop />,
  stopped: <Icon.Play />,
  stopping: <Icon.Loading />,
  starting: <Icon.Loading />,
};

export const LEVEL_COLORS: Record<string, string> = {
  info: "var(--pluto-gray-l8)",
  error: "var(--pluto-error-z)",
  fatal: "var(--pluto-error-z)",
  warn: "var(--pluto-warning-m1)",
};
