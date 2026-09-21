// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";

import { Command } from "@/app/command";
import { createConsoleWrapper } from "@/testutil";

const Palette = (): ReactElement => {
  const [value, setValue] = useState(">");
  return (
    <Command.List
      value={value}
      onChange={setValue}
      inputPlaceholder={<>Type &gt; to view commands</>}
    />
  );
};

/** Renders the command list of the app and filters it by the given text. */
export const renderCommands = async (text: string): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: null });
  render(<Palette />, { wrapper });
  const input = await waitFor(() => screen.getByRole("textbox"));
  fireEvent.change(input, { target: { value: `>${text}` } });
};
