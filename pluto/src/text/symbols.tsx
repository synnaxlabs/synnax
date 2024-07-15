// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { runtime } from "@synnaxlabs/x";
import { type FC } from "react";

const isWindows = runtime.getOS() === "Windows";

export const Symbols: Record<string, FC> = {
  Meta: isWindows ? Icon.Keyboard.Control : Icon.Keyboard.Command,
  Alt: isWindows ? () => "Alt" : Icon.Keyboard.Option,
};
