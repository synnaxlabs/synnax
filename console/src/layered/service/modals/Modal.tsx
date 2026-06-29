// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layered/app/modals/Modal.css";

import { Dialog, Errors } from "@synnaxlabs/pluto";
import { memo, type ReactElement, useCallback } from "react";

import { useStore } from "@/layered/session/modals/Provider";
import { type Entry } from "@/layered/session/modals/store";

interface ModalProps {
  entry: Entry;
}

export const Modal = memo(({ entry }: ModalProps): ReactElement => {
  const { key, Renderer, params } = entry;
  const store = useStore();
  const dismiss = useCallback(() => store.close(key), [store, key]);
  const close = useCallback(
    (result?: unknown) => store.close(key, result),
    [store, key],
  );
  return (
    <Dialog.Frame
      key={key}
      variant="modal"
      visible
      onVisibleChange={dismiss}
      background={0}
    >
      <Errors.SuspenseBoundary>
        <Renderer params={params} close={close} />
      </Errors.SuspenseBoundary>
    </Dialog.Frame>
  );
});
Modal.displayName = "Modal";
