// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Errors } from "@synnaxlabs/pluto";
import { memo, type ReactElement, useCallback } from "react";

import { type Entry } from "@/layered/session/modals/Context";

interface ModalProps extends Entry {}

export const Modal = memo(({ dismiss, render }: ModalProps): ReactElement => {
  const handleDismiss = useCallback(() => dismiss(), [dismiss]);
  return (
    <Dialog.Frame
      variant="modal"
      visible
      onVisibleChange={handleDismiss}
      background={0}
    >
      <Errors.SuspenseBoundary>{render()}</Errors.SuspenseBoundary>
    </Dialog.Frame>
  );
});
Modal.displayName = "Modal";
