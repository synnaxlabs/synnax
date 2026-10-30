// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/keys/Keys.css";

import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type text } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";

export interface KeysProps extends ComponentPropsWithoutRef<"span"> {
  trigger: Triggers.Trigger;
  level?: text.Level;
}

/**
 * Renders a shortcut in the keys of the reader's platform. Astro renders the docs to
 * static HTML at build time, where no browser can report the platform, so Triggers.Text
 * alone would print macOS keys for every reader. The page holds the keys of macOS and
 * of Windows, and the data-os attribute on the root element shows one set. In running
 * text, the chip sizes with the text.
 */
export const Keys = ({
  trigger,
  level,
  className,
  ...rest
}: KeysProps): ReactElement => (
  <span
    className={className == null ? "docs-keys" : `docs-keys ${className}`}
    {...rest}
  >
    <Triggers.Text
      el="span"
      trigger={trigger}
      level={level}
      os="macOS"
      className="docs-keys__mac"
    />
    <Triggers.Text
      el="span"
      trigger={trigger}
      level={level}
      os="Windows"
      className="docs-keys__pc"
    />
  </span>
);
