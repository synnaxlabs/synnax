// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { memo, type ReactElement, useEffect } from "react";

import { Errors } from "@/platform/errors";
import { type Session } from "@/session";

const [LockContext, useSetLocked] = context.create<(locked: boolean) => void>({
  displayName: "Modals.LockContext",
  providerName: "Modals.Stack",
});

/**
 * useLock blocks the user from dismissing the enclosing modal while locked is true. The
 * modal can still close itself, and clearing the modal store still dismisses it.
 */
export const useLock = (locked: boolean): void => {
  const setLocked = useSetLocked("Modals.useLock");
  useEffect(() => setLocked(locked), [setLocked, locked]);
};

interface ModalProps extends Omit<Session.Modals.Entry, "key"> {}

export const Modal = memo(
  ({ dismiss, render, locked, setLocked }: ModalProps): ReactElement => {
    const handleDismiss = () => {
      if (!locked) dismiss();
    };
    return (
      <Dialog.Frame
        variant="modal"
        visible
        onVisibleChange={handleDismiss}
        background={0}
      >
        <LockContext value={setLocked}>
          <Errors.SuspenseBoundary>{render()}</Errors.SuspenseBoundary>
        </LockContext>
      </Dialog.Frame>
    );
  },
);
Modal.displayName = "Modal";
