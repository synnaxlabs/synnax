// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type PropsWithChildren, type ReactElement } from "react";

export interface FeedbackButtonProps extends PropsWithChildren {
  initialVisible?: boolean;
}

/**
 * Renders the feedback button and dialog. The page renders it closed and empty, and the
 * feedback island mounts it open with the form as children.
 */
export const FeedbackButton = ({
  initialVisible,
  children,
}: FeedbackButtonProps): ReactElement => (
  <Dialog.Frame
    className="feedback-modal"
    variant="modal"
    initialVisible={initialVisible}
  >
    <Dialog.Trigger
      className="feedback-button"
      size="medium"
      gap="small"
      variant="outlined"
    >
      <Icon.Feedback />
      Stuck? Let us know!
    </Dialog.Trigger>
    <Dialog.Dialog>{children}</Dialog.Dialog>
  </Dialog.Frame>
);
