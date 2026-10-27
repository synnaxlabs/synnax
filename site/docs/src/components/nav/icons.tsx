// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

/** The icon of each section of the reference, by the key of the section's nav group. */
export const SECTION_ICONS: Record<string, ReactElement> = {
  concepts: <Icon.Reference />,
  core: <Icon.Core />,
  client: <Icon.Terminal />,
  control: <Icon.Control />,
  console: <Icon.Dashboard />,
  driver: <Icon.Device />,
};
