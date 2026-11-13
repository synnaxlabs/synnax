// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/lyra/triggers";
import { Panel } from "@synnaxlabs/pluto";
import { memo, type PropsWithChildren, type ReactElement } from "react";

import { Modal } from "@/platform/modals/Modal";
import { Session } from "@/session";

const CLOSE_TRIGGERS: Triggers.Trigger[] = [Panel.CLOSE_TRIGGER];

interface LayerProps {
  entry: Session.Modals.Entry;
  // From the rendered stack, so a modal that closes during a key press does not pass
  // the rest of that press to the modal below.
  top: boolean;
}

const Layer = memo(({ entry, top }: LayerProps): ReactElement => {
  const { key, ...rest } = entry;
  return (
    <Triggers.Scope active={top}>
      <Modal key={key} {...rest} />
    </Triggers.Scope>
  );
});
Layer.displayName = "Layer";

/**
 * Renders the open modals over its children. Only the top modal hears triggers, and the
 * close trigger closes it.
 */
export const Stack = ({ children }: PropsWithChildren): ReactElement => {
  const stack = Session.Modals.useStack();
  const store = Session.Modals.useStore("Modals.Stack");
  Triggers.use({
    triggers: CLOSE_TRIGGERS,
    loose: true,
    // Above the page's close trigger, which would close a tab once the modal is gone.
    priority: 1,
    callback: ({ stage, stopPropagation }) => {
      if (stage !== "start" || !store.isAnyOpen()) return;
      store.closeTop();
      stopPropagation();
    },
  });
  return (
    <>
      {stack.map((entry, i) => (
        <Layer key={entry.key} entry={entry} top={i === stack.length - 1} />
      ))}
      <Triggers.Scope active={() => !store.isAnyOpen()}>{children}</Triggers.Scope>
    </>
  );
};
