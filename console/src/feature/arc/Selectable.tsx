// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { Arc } from "@/platform/arc";
import { Selector } from "@/platform/selector";

export const Selectable = Selector.createSelectable({
  title: "Arc Automation",
  icon: <Icon.Arc />,
  useOnSelect: Arc.useCreate,
  type: arc.TYPE_ONTOLOGY_ID.type,
  useVisible: () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID),
});
