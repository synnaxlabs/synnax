// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type capture } from "@/index";

import { create, PARK, setup } from "./common/calculated-channel";

/**
 * Docs `client/calculated-channels/create`: click the create calculated channel button
 * in the Channels toolbar, name the channel, write an expression over a live metrics
 * channel, and click Create. The new channel lands in the toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const modal = await setup(session);

  session.startRecording();
  await session.hold(1500);

  await create(session, modal);
  await session.moveTo(PARK);
  await session.hold(3000);
};
