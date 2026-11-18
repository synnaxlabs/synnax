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
import { memo, type ReactElement, useLayoutEffect } from "react";

import { Errors } from "@/platform/errors";
import { type Session } from "@/session";

const [BusyContext, useSetBusy] = context.create<Session.Modals.Entry["setBusy"]>({
  displayName: "Modals.BusyContext",
  defaultValue: () => {},
});

/**
 * Keeps the modal open while busy is true: Escape, the close trigger, and a click
 * outside do nothing. Call it with the modal's save state, and disable its cancel
 * button with the same value. Does nothing outside a modal.
 */
export const useBusy = (busy: boolean): void => {
  const setBusy = useSetBusy();
  // A layout effect marks the modal busy before the browser handles another key press.
  useLayoutEffect(() => {
    if (!busy) return;
    setBusy(true);
    return () => setBusy(false);
  }, [busy, setBusy]);
};

interface ModalProps extends Session.Modals.Entry {}

export const Modal = memo(({ dismiss, render, setBusy }: ModalProps): ReactElement => {
  const handleDismiss = () => dismiss();
  return (
    <Dialog.Frame
      variant="modal"
      visible
      onVisibleChange={handleDismiss}
      background={0}
    >
      <BusyContext value={setBusy}>
        <Errors.SuspenseBoundary>{render()}</Errors.SuspenseBoundary>
      </BusyContext>
    </Dialog.Frame>
  );
});
Modal.displayName = "Modal";
