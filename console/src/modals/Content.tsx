// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type Spec } from "@/modals/Modals";
import { useRenderer } from "@/modals/renderer";

export interface ContentProps {
  spec: Spec;
  onClose: (result?: unknown) => void;
}

/** Renders the registered renderer for the active modal. */
export const Content = ({ spec, onClose }: ContentProps): ReactElement => {
  const Renderer = useRenderer(spec.type);
  return (
    <Errors.SuspenseBoundary>
      <Renderer
        key={spec.key}
        layoutKey={spec.key}
        onClose={onClose}
        visible
        focused
        active
      />
    </Errors.SuspenseBoundary>
  );
};
