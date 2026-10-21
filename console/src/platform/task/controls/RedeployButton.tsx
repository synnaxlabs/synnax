// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface RedeployButtonProps extends Omit<Button.ButtonProps, "onClick"> {
  /** Reveals the button when true and collapses it when false. */
  visible: boolean;
  /** Click handler */
  onClick: () => void;
  /** Replaces the tooltip's generic reason for the redeploy. One line each. */
  reasons?: string[];
}

/**
 * Deploys the saved configuration to the running task. Stays mounted so its reveal and
 * collapse can animate; hidden it occupies no space and cannot be focused or clicked.
 */
export const RedeployButton = ({
  visible,
  onClick,
  reasons = ["Configuration changed since deploy."],
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
        tooltip={
          <>
            {reasons.map((line) => (
              <Text.Text key={line} level="small" color={11}>
                {line}
              </Text.Text>
            ))}
            <Text.Text
              className={CSS.BE("task-redeploy", "apply")}
              level="small"
              color={11}
            >
              Redeploy to apply.
            </Text.Text>
          </>
        }
        variant="filled"
        {...props}
      >
        <Icon.Refresh />
        Redeploy
      </Button.Button>
    </div>
  </div>
);
