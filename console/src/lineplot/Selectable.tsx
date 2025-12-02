// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { create, LAYOUT_TYPE } from "@/lineplot/layout";
import { type Selector } from "@/selector";

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Line Plot",
  icon: <Icon.LinePlot />,
  useVisible: () => Access.useEditGranted(lineplot.ontologyID("")),
  create: async ({ layoutKey }) => create({ key: layoutKey }),
};
