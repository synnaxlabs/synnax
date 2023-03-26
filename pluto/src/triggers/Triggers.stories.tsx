// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentMeta } from "@storybook/react";

import { Triggers } from ".";

const story: ComponentMeta<typeof Triggers.Provider> = {
  title: "Triggers/Triggers",
  component: Triggers.Provider,
};

export const Basic = (): JSX.Element => {
  return (
    <Triggers.Provider>
      <Child />
    </Triggers.Provider>
  );
};

const Child = (): JSX.Element => {
  const { triggers } = Triggers.useHeld({
    triggers: [
      ["ArrowDown"],
      ["ArrowUp", "Shift"],
      ["MouseLeft", "Alt"],
      ["Shift", "ArrowDown"],
      ["Shift"],
      ["MouseLeft", "MouseLeft"],
    ],
    loose: true,
  });
  return (
    <div>
      {triggers.map((trigger, i) => (
        <h1 key={i}>{Array.isArray(trigger) ? trigger.join(" + ") : trigger}</h1>
      ))}
    </div>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
