// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export const CreateIcon = (): ReactElement => (
  <PIcon.Create>
    <Icon.Table />
  </PIcon.Create>
);

export const ImportIcon = (): ReactElement => (
  <PIcon.Import>
    <Icon.Table />
  </PIcon.Import>
);
