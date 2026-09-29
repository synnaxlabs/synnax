// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arinc429 } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { Select } from "@/feature/arinc429/device/device";
import { Bus } from "@/feature/bus";

const checkIdentifier = Bus.checkIdentifier("arinc429", "ARINC 429");

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
  prefix: "arinc429",
  name: "ARINC 429",
  icon: <Icon.Wave.Square />,
  readConfigZ: arinc429.readConfigZ,
  writeConfigZ: arinc429.writeConfigZ,
  accepts: (entry) => checkIdentifier(entry) == null,
  SelectDevice: Select,
  readMessageChecks: [checkIdentifier],
  writeMessageChecks: [checkIdentifier],
});
