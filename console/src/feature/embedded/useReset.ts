// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { TimeSpan } from "@synnaxlabs/x";

import { reset } from "@/feature/embedded/supervisor";
import { License } from "@/platform/license";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

const HOLD_TO_CONFIRM = TimeSpan.seconds(1);

/**
 * Returns a callback that erases everything Synnax Desktop has stored, logs the machine
 * out, and starts the app again. It asks first, and the person must hold the button to
 * confirm.
 */
export const useReset = (): (() => void) => {
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const store = Session.useStore();
  return () =>
    handleError(async () => {
      const confirmed = await confirm({
        message: "Are you sure you want to erase all data?",
        description:
          "This permanently deletes every channel, all recorded data, and every project, logs you out, and then restarts Synnax. It cannot be undone. Hold the button to confirm.",
        confirm: { label: "Erase all data", delay: HOLD_TO_CONFIRM.milliseconds },
      });
      if (confirmed !== true) return;
      addStatus({
        variant: "info",
        message: "Erasing all data",
        description: "Synnax restarts when the erase finishes.",
      });
      // An erase works offline. A machine the hub never hears from leaves its account
      // when its license expires.
      const { link } = Session.Account.selectSliceState(store.getState());
      if (link != null) await License.unlink(link.secret).catch(console.error);
      store.dispatch(Session.Account.clear());
      // The restart drops any change still waiting on the debounce.
      await Session.Persist.flush(store.dispatch);
      await reset();
    }, "Failed to erase data");
};
