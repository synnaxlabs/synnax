// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { parseDesktopInvite } from "@/platform/core/invite";
import { Session } from "@/session";

/**
 * Returns a function that takes a desktop invite link and fills in its code for the
 * saved Core with the link's address.
 * @throws {Error} when the link has no Core address or no invite code, or when no
 * saved Core has its address. A link never adds a Core, so it cannot aim a login form
 * at a server the person did not choose.
 */
export const useReceiveDesktopInvite = (): ((link: string) => void) => {
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  return (link) => {
    const { code, host, port, secure } = parseDesktopInvite(link);
    const saved = Object.values(store.getState()[Session.Core.SLICE_NAME].cores).find(
      (c) => c.host === host && c.port === port && c.secure === secure,
    );
    if (saved == null)
      throw new Error(
        `No saved Core is at ${host}:${port}. Connect a Core at that address, then open the link again.`,
      );
    dispatch(Session.Link.receiveInvite({ core: saved.key, code }));
  };
};
