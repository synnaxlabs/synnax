// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Button, Text } from "@synnaxlabs/pluto";
import { UseMutationResult } from "@tanstack/react-query";
import { type ReactElement } from "react";

export interface ConfirmProps {
  confirm: UseMutationResult<void, Error, void, unknown>;
  progress?: string;
}

export const Confirm = ({
  confirm: { isPending, isSuccess, mutate },
  progress,
}: ConfirmProps): ReactElement => {
  return (
    <Align.Center>
      <Align.Space
        style={{
          maxWidth: 600,
          padding: "20rem 20rem",
          borderRadius: "1rem",
          backgroundColor: "var(--pluto-gray-l1)",
        }}
        bordered
        rounded
        align="center"
        size={10}
      >
        <Text.Text level="h1">Ready to go?</Text.Text>
        <Text.Text level="p">
          Once you click "Confirm" Hitting confirm will make permanent changes to the
          channels in your Synnax cluster. To edit information in the previous steps,
          you'll need to reconfigure the device. Hit confirm to proceed.
        </Text.Text>
        {isPending && <Text.Text level="p">{progress}</Text.Text>}
        <Button.Button
          onClick={() => mutate()}
          loading={isPending}
          disabled={isPending || isSuccess}
        >
          {isSuccess ? "Success!" : "Confirm"}
        </Button.Button>
      </Align.Space>
    </Align.Center>
  );
};
