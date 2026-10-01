// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { useCallback, useState } from "react";

import { createPrompt } from "@/platform/modals/factory";
import { Triggers } from "@/platform/triggers";
import { type Session } from "@/session";

export interface RenameParams {
  allowEmpty?: boolean;
  initialValue?: string;
  label?: string;
  title?: string;
  icon?: Icon.ReactElement;
}

const Rename = ({
  allowEmpty = false,
  label = "Name",
  title = "Name",
  initialValue = "",
  icon,
  close,
}: Session.Modals.ContentProps<RenameParams, string>) => {
  const [name, setName] = useState(initialValue);
  const [error, setError] = useState<string | undefined>(undefined);
  const handleClick = useCallback(() => {
    if (allowEmpty && name.length === 0) return close();
    if (!allowEmpty && name.length === 0) return setError(`${label} is required`);
    return close(name);
  }, [close, setError, name, allowEmpty]);
  return (
    <Modal.Frame>
      <Modal.Header icon={icon}>{title}</Modal.Header>
      <Modal.Body>
        <Input.Item
          label={label}
          required={!allowEmpty}
          helpText={error}
          status={error != null ? "error" : "success"}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder={label}
            level="h2"
            variant="text"
            value={name}
            onChange={setName}
            selectOnFocus
          />
        </Input.Item>
      </Modal.Body>
      <Modal.Footer>
        <Triggers.SaveHelpText action="Save" trigger={Triggers.SAVE} />
        <Nav.Bar.End x align="center">
          <Button.Button
            status="success"
            disabled={!allowEmpty && name.length === 0}
            variant="filled"
            onClick={handleClick}
            trigger={Triggers.SAVE}
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useRename = createPrompt<string, RenameParams>(Rename);
