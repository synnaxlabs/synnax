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
import { type ReactElement } from "react";

import { Form } from "@/components/feedback/Form";

export interface ButtonProps {
  initialVisible?: boolean;
}

/**
 * Renders the feedback button and dialog. The page renders it closed, which also puts
 * the form's styles in the page, and the feedback island mounts it open.
 */
export const Button = ({ initialVisible }: ButtonProps): ReactElement => (
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
    <Dialog.Dialog>
      <Form />
    </Dialog.Dialog>
  </Dialog.Frame>
);
