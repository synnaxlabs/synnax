// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Errors } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";

import { type Entry } from "@/layered/session/modals/store";

interface ModalProps {
  entry: Entry;
}

export const Modal = memo(
  ({ entry }: ModalProps): ReactElement => (
    <Dialog.Frame
      variant="modal"
      visible
      onVisibleChange={entry.dismiss}
      background={0}
    >
      <Errors.SuspenseBoundary>{entry.render()}</Errors.SuspenseBoundary>
    </Dialog.Frame>
  ),
);
Modal.displayName = "Modal";
