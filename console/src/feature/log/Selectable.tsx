// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { Log } from "@/platform/log";
import { Selector } from "@/platform/selector";

export const Selectable = Selector.createSelectable({
  type: log.TYPE_ONTOLOGY_ID.type,
  title: "Log",
  icon: <Icon.Log />,
  useOnSelect: Log.useCreate,
  useVisible: () => Access.useCreateGranted(log.TYPE_ONTOLOGY_ID),
});
