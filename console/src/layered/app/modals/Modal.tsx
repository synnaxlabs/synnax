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
import { type CSSProperties, type ReactElement, useCallback } from "react";

import { DismissProvider } from "@/layered/service/modals/Dismiss";
import { useStore } from "@/layered/session/modals/Provider";
import { type Entry, type Size } from "@/layered/session/modals/store";

const sizeCSS = (size?: Size): CSSProperties => ({
  maxWidth: size?.width,
  maxHeight: size?.height,
});

interface ModalProps {
  entry: Entry;
}

export const Modal = ({ entry }: ModalProps): ReactElement => {
  const { key, render: Renderer, size, args } = entry;
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
      <Dialog.Dialog style={sizeCSS(size)} full>
        <Errors.SuspenseBoundary>
          <DismissProvider value={dismiss}>
            <Renderer args={args} close={close} />
          </DismissProvider>
        </Errors.SuspenseBoundary>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
