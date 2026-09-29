// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { can } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { Bus } from "@/feature/bus";
import { Select } from "@/feature/can/device/device";

export const {
  PREFIX,
  READ_TYPE,
  WRITE_TYPE,
  READ_SCHEMAS,
  WRITE_SCHEMAS,
  Read,
  Write,
  useCreateRead,
  useCreateWrite,
  COMMANDS,
  SELECTABLES,
  FORMS,
} = Bus.createTasks({
  prefix: "can",
  name: "CAN",
  icon: <Icon.Hardware />,
  readConfigZ: can.readConfigZ,
  writeConfigZ: can.writeConfigZ,
  accepts: ({ identifier }) => identifier?.type === "can",
  SelectDevice: Select,
});
