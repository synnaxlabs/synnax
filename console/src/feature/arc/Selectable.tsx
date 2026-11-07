// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { Arc } from "@/platform/arc";
import { type Selector } from "@/platform/selector";

export const Selectable: Selector.Selectable = {
  title: "Arc automation",
  icon: <Icon.Arc />,
  description: "Automate control",
  useOnSelect: Arc.useCreate,
  type: arc.TYPE_RESOURCE_ID.type,
  useVisible: () => Access.useCreateGranted(arc.TYPE_RESOURCE_ID),
};
