// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";

import { Triggers } from "@/platform/triggers";

export interface ConnectFooterProps {
  /** The result of {@link useBlockedReason}. A reason blocks Connect. */
  reason: string | null;
  variant: status.Variant;
  message?: string;
  onConnect: () => void;
  onBack?: () => void;
}

export const ConnectFooter = ({
  reason,
  variant,
  message,
  onConnect,
  onBack,
}: ConnectFooterProps): ReactElement => {
  let summary: ReactElement;
  if (reason != null) summary = <Status.Summary variant="warning" message={reason} />;
  else if (variant === "success")
    summary = <Triggers.SaveHelpText action="Connect" noBar />;
  else summary = <Status.Summary variant={variant} message={message} />;
  return (
    <Modal.Footer>
      <Nav.Bar.Start gap="small">
        {onBack != null && (
          <Button.Button variant="text" onClick={onBack}>
            <Icon.Arrow.Left />
            Back
          </Button.Button>
        )}
        {summary}
      </Nav.Bar.Start>
      <Nav.Bar.End>
        <Button.Button
          status={status.keepVariants(variant, "loading")}
          onClick={onConnect}
          variant="filled"
          trigger={Triggers.SAVE}
          disabled={reason != null}
        >
          Connect
        </Button.Button>
      </Nav.Bar.End>
    </Modal.Footer>
  );
};
