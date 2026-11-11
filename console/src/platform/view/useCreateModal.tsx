// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { type ReactElement, useState } from "react";

import { Appearance } from "@/platform/appearance";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

/** The name, icon, and color of a view to create. */
export interface CreateResult extends Pick<view.New, "name" | "icon" | "color"> {}

const Create = ({
  close,
}: Modals.ContentProps<Record<never, never>, CreateResult>): ReactElement => {
  const [name, setName] = useState("");
  const [icon, setIcon] = useState<string>();
  const [color, setColor] = useState<view.New["color"]>();
  const trimmed = name.trim();
  const handleCreate = () => {
    if (trimmed.length > 0) close({ name: trimmed, icon, color });
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.View />}>View.Create</Modal.Header>
      <Modal.Body>
        <Flex.Box x align="center" gap="medium">
          <Dialog.Frame zIndex={100}>
            <Dialog.Trigger
              hideCaret
              variant="outlined"
              size="large"
              tooltip="Change icon"
              aria-label="Change icon"
            >
              {Appearance.renderIcon({ icon, color, fallback: Icon.View })}
            </Dialog.Trigger>
            <Appearance.Dialog aria-label="View icon">
              <Appearance.Picker
                icon={icon}
                color={color}
                fallback={Icon.View}
                onIconChange={setIcon}
                onColorChange={setColor}
              />
            </Appearance.Dialog>
          </Dialog.Frame>
          <Input.Text
            placeholder="View name"
            aria-label="View name"
            variant="text"
            autoFocus
            level="h3"
            value={name}
            onChange={setName}
          />
        </Flex.Box>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            disabled={trimmed.length === 0}
            onClick={handleCreate}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/** Asks for the name, icon, and color of a view to create. */
export const useCreateModal = Modals.createPrompt<CreateResult>(Create);
