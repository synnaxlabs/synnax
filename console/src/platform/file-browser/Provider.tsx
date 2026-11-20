// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax as Client } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { Session } from "@/platform/file-browser/session";

class Sessions {
  readonly client: Client | null;
  private readonly byRack = new Map<rack.Key, Session>();

  constructor(client: Client | null) {
    this.client = client;
  }

  get(rack: rack.Key): Session | null {
    if (this.client == null) return null;
    let session = this.byRack.get(rack);
    if (session == null) {
      session = new Session(this.client, rack);
      this.byRack.set(rack, session);
    }
    return session;
  }
}

const [Context, useContext] = context.create<Sessions>({
  displayName: "FileBrowser.Context",
  providerName: "FileBrowser.Provider",
});

/**
 * Keeps a file session for the Driver of each rack, so a listing outlives the dialog
 * that showed it. A new Core connection starts new sessions.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const client = Synnax.use();
  const [sessions, setSessions] = useState(() => new Sessions(client));
  if (sessions.client !== client) setSessions(new Sessions(client));
  return <Context value={sessions}>{children}</Context>;
};

/** @returns the file session of the rack's Driver, or null with no Core connection. */
export const useSession = (rack: rack.Key): Session | null =>
  useContext("FileBrowser.useSession").get(rack);
