// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Access, User as PUser } from "@synnaxlabs/pluto";

import { Palette } from "@/palette";
import { User } from "@/user";

const RegisterCommand = Palette.createSimpleCommand({
  key: "register-user",
  name: "Register a user",
  icon: <PUser.CreateIcon />,
  layout: User.REGISTER_LAYOUT,
  useVisible: () => Access.useCreateGranted(user.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [RegisterCommand];
