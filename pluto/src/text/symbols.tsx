// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { runtime } from "@synnaxlabs/x";

const isWindows = runtime.getOS() === "Windows";

export const Symbols = {
  Meta: isWindows ? Icon.Keyboard.Control : Icon.Keyboard.Command,
  Alt: isWindows ? Icon.Keyboard.Alt : Icon.Keyboard.Option,
};
