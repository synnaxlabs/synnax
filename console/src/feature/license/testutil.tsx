// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { screen } from "@testing-library/react";

import { Guard } from "@/feature/license/Guard";
import { renderWithStatus, type StatusHarness } from "@/testutil";

export const GUARDED_CONTENT = "licensed content";

/** Matches the handoff code that the Console puts in a portal link. */
export const HANDOFF_CODE = /^[0-9a-f-]{36}$/;

/**
 * Renders a {@link Guard} over {@link GUARDED_CONTENT} with the given client and
 * connection status. The client stays unconnected, because an unlicensed Core never
 * settles a connection.
 */
export const renderGuard = async (
  client: Client | null,
  status?: connection.Status,
): Promise<StatusHarness> =>
  await renderWithStatus(
    <Guard>
      <span>{GUARDED_CONTENT}</span>
    </Guard>,
    { client, status },
  );

/** Finds the button labeled `label` that opens the portal, and returns its link. */
export const findPortalLink = async (label: string): Promise<string> => {
  const link = (await screen.findByText(label)).closest("a");
  if (link == null) throw new Error(`${label} is not a link`);
  return link.href;
};
