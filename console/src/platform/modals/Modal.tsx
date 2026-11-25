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
import { Modal as Base } from "@synnaxlabs/lyra/modal";
import { Errors as PErrors } from "@synnaxlabs/pluto";
import { type ReactElement, useLayoutEffect } from "react";

import { Errors } from "@/platform/errors";
import { useConfirm } from "@/platform/modals/useConfirm";
import type { Session } from "@/session";

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

const [UnsavedContext, useSetUnsaved] = context.create<(unsaved: boolean) => void>({
  displayName: "Modals.UnsavedContext",
  defaultValue: () => {},
});

/**
 * While unsaved is true, Escape, the close trigger, and a click outside ask the user to
 * discard before the modal closes. A close that the content calls, as after a save,
 * does not ask. Does nothing outside a modal.
 */
export const useUnsaved = (unsaved: boolean): void => {
  const setUnsaved = useSetUnsaved();
  useLayoutEffect(() => {
    if (!unsaved) return;
    setUnsaved(true);
    return () => setUnsaved(false);
  }, [unsaved, setUnsaved]);
};

// The content draws its own surface, so a fallback in its place needs one too.
const Fallback = (props: PErrors.FallbackProps): ReactElement => (
  <Base.Frame>
    <Errors.Diagnostics {...props} />
  </Base.Frame>
);

interface ModalProps extends Session.Modals.Entry {}

export const Modal = ({
  dismiss,
  render,
  setBusy,
  setGuard,
}: ModalProps): ReactElement => {
  const confirm = useConfirm();
  const confirmDiscard = async () =>
    (await confirm({
      message: "Discard unsaved edits?",
      description: "If you close now, you lose the edits since the last save.",
      confirm: { label: "Discard" },
      cancel: { label: "Keep editing" },
      title: "Unsaved edits",
    })) === true;
  const setUnsaved = (unsaved: boolean) =>
    setGuard(unsaved ? confirmDiscard : undefined);
  const handleDismiss = () => dismiss();
  return (
    <Dialog.Frame
      variant="modal"
      visible
      onVisibleChange={handleDismiss}
      background={0}
    >
      <BusyContext value={setBusy}>
        <UnsavedContext value={setUnsaved}>
          <PErrors.SuspenseBoundary FallbackComponent={Fallback}>
            {render()}
          </PErrors.SuspenseBoundary>
        </UnsavedContext>
      </BusyContext>
    </Dialog.Frame>
  );
};
Modal.displayName = "Modal";
