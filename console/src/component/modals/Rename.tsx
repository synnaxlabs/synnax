// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, type Icon, Input, Nav } from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { Body } from "@/component/modals/Body";
import { createPrompt, type Prompt } from "@/component/modals/factory";
import { Footer } from "@/component/modals/Footer";
import { Frame } from "@/component/modals/Frame";
import { Header } from "@/component/modals/Header";
import { type ContentProps } from "@/session/modals/Context";
import { Triggers } from "@/component/triggers";

export interface RenameParams {
  allowEmpty?: boolean;
  initialValue?: string;
  label?: string;
  title?: string;
  icon?: Icon.ReactElement;
}

export interface PromptRename extends Prompt<string, RenameParams> {}

const Rename = ({
  allowEmpty = false,
  label = "Name",
  title = "Name",
  initialValue = "",
  icon,
  close,
}: ContentProps<RenameParams, string>) => {
  const [name, setName] = useState(initialValue);
  const [error, setError] = useState<string | undefined>(undefined);
  const handleClick = useCallback(() => {
    if (allowEmpty && name.length === 0) return close();
    if (!allowEmpty && name.length === 0) return setError(`${label} is required`);
    return close(name);
  }, [close, setError, name, allowEmpty]);
  return (
    <Frame>
      <Header icon={icon}>{title}</Header>
      <Body>
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
      </Body>
      <Footer>
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
      </Footer>
    </Frame>
  );
};

export const useRename = createPrompt<string, RenameParams>(Rename);
