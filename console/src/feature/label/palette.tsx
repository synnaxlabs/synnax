// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { useEditModal } from "@/platform/label/useEditModal";
import { Palette } from "@/platform/palette";

export const EditCommand = Palette.createCommand({
  key: "edit_labels",
  name: "Edit labels",
  icon: <Icon.Label />,
  useOnSelect: useEditModal,
  useVisible: () => Access.useUpdateGranted(label.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [EditCommand];
