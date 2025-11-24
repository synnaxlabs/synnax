// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, User as PUser } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { User } from "@/user";

const REGISTER_USER_COMMAND: Palette.Command = {
  icon: <Icon.User />,
  name: "Register a User",
  key: "register-user",
  onSelect: ({ placeLayout }) => placeLayout(User.REGISTER_LAYOUT),
  visible: ({ store, client }) => PUser.editAccessGranted({ key: "", store, client }),
};

export const COMMANDS = [REGISTER_USER_COMMAND];
