// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Mosaic.css";

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Logo } from "@synnaxlabs/media";
import { Eraser, Flex, Text, Triggers } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";

import { Layout } from "@/layout";
import { useSelectorVisible } from "@/layouts/Selector";
import { Panel } from "@/panel";

const EmptyContent = (): ReactElement => {
  const createComponentEnabled = useSelectorVisible();
  return (
    <Eraser.Eraser>
      <Flex.Box gap={5} center>
        <Logo className="synnax-logo-watermark" />
        {createComponentEnabled && (
          <Flex.Box x gap="small">
            <Text.Text level="h5" weight={450} color={9}>
              New Component
            </Text.Text>
            <Flex.Box x empty>
              <Triggers.Text level="h5" trigger={["Control", "T"]} />
            </Flex.Box>
          </Flex.Box>
        )}
      </Flex.Box>
    </Eraser.Eraser>
  );
};
export const MOSAIC_LAYOUT_TYPE = "mosaic";

// Mosaic renders the active panel for the current window. The legacy per-window
// Redux mosaic is gone: tiling is now a server-backed panel read through Flux. When
// no panel is active (no project, or a project with nothing open yet) the empty
// state is shown.
//
// TODO(panels): re-wire onto the panel host the drop integrations the legacy mosaic
// owned — file-drop import (Import.dataTransferItem) and per-type ontology
// onMosaicDrop. The panel host already creates resource tabs from dropped ontology
// IDs; these richer drops are not yet ported.
export const Mosaic = memo((): ReactElement | null => {
  const windowKey = useSelectWindowKey();
  const activePanelKey = Layout.useSelectActivePanelKey();
  if (windowKey == null || activePanelKey == null) return <EmptyContent />;
  return <Panel.Mosaic panelKey={activePanelKey} windowKey={windowKey} />;
});
Mosaic.displayName = "Mosaic";
