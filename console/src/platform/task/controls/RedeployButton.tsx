// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface RedeployButtonProps extends Omit<Button.ButtonProps, "onClick"> {
  /** Reveals the button when true and collapses it when false. */
  visible: boolean;
  /** Click handler */
  onClick: () => void;
}

/**
 * Deploys the saved configuration to the running task. Stays mounted so its reveal and
 * collapse can animate; hidden it occupies no space and cannot be focused or clicked.
 */
export const RedeployButton = ({
  visible,
  onClick,
  ...props
}: RedeployButtonProps): ReactElement => (
  <div
    className={CSS.cls(
      CSS.B("task-redeploy"),
      visible && CSS.BM("task-redeploy", "visible"),
    )}
    aria-hidden={!visible}
  >
    <div className={CSS.BE("task-redeploy", "clip")}>
      <Button.Button
        onClick={onClick}
        size="medium"
        tabIndex={visible ? undefined : -1}
        tooltip="Configuration changed since deploy. Redeploy to apply."
        variant="filled"
        {...props}
      >
        <Icon.Refresh />
        Redeploy
      </Button.Button>
    </div>
  </div>
);
