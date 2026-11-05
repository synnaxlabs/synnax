// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { video } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { useCreate } from "@/feature/video/useCreate";
import { Selector } from "@/platform/selector";

export const Selectable = Selector.createSelectable({
  type: video.TYPE_ONTOLOGY_ID.type,
  title: "Video",
  icon: <Icon.Video />,
  useOnSelect: useCreate,
  useVisible: () => Access.useCreateGranted(video.TYPE_ONTOLOGY_ID),
});
