// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { retrieveAndPlaceLayout } from "@/hardware/task/layouts";
import { type Link } from "@/link";

export const handleLink: Link.Handler = async ({ client, key, placeLayout }) =>
  await retrieveAndPlaceLayout(client, key, placeLayout);
