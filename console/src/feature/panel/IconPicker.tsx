// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/panel/IconPicker.css";

import { panel } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Errors, Panel } from "@synnaxlabs/pluto";
import { type color } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { renderIcon } from "@/feature/panel/icon";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { type Session } from "@/session";

export interface IconPickerParams {
  /** The panel whose icon and color the picker sets. */
  panel: panel.Key;
}

type ContentProps = Session.Modals.ContentProps<IconPickerParams, void>;

// A color pick keeps the picker open so the user can see it on each icon. An icon pick
// closes it.
const Content = ({ panel: key, close }: ContentProps): ReactElement => {
  Panel.useEnsure({ key });
  const icon = Panel.useIcon({ key });
  const color = Panel.useColor({ key });
  const { colors } = Theming.use();
  const { dispatch } = Panel.useDispatch();
  const updateFrequent = Color.useFrequentUpdater();
  const handleColorChange = useCallback(
    (next?: color.Color) => {
      dispatch({ key, actions: panel.setColor({ color: next }) });
      if (next != null) updateFrequent(next);
    },
    [dispatch, key, updateFrequent],
  );
  const handleIconChange = useCallback(
    (next: string) => {
      dispatch({ key, actions: panel.setIcon({ icon: next }) });
      close();
    },
    [dispatch, key, close],
  );
  const handleReset = useCallback(() => {
    dispatch({ key, actions: [panel.setIcon({}), panel.setColor({})] });
    close();
  }, [dispatch, key, close]);
  return (
    <Modal.Frame className={CSS.B("panel-icon-picker")}>
      <Modal.Header icon={renderIcon({ icon, color })}>Panel icon</Modal.Header>
      <Modal.Body gap="large">
        <Color.Swatches
          value={color}
          onChange={handleColorChange}
          fallback={colors.gray.l10}
        />
        <Input.IconPicker value={icon} onChange={handleIconChange} color={color} />
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="text"
            onClick={handleReset}
            disabled={icon == null && color == null}
          >
            Reset
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

const Boundary = (props: ContentProps): ReactElement => (
  <Errors.SuspenseBoundary>
    <Content {...props} />
  </Errors.SuspenseBoundary>
);

export const useIconPicker = Modals.create<IconPickerParams>(Boundary);
