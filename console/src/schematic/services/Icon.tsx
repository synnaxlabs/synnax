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

export const CreateIcon = () => (
  <PIcon.Create>
    <Icon.Schematic />
  </PIcon.Create>
);

export const ImportIcon = () => (
  <PIcon.Import>
    <Icon.Schematic />
  </PIcon.Import>
);
