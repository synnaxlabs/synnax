import { type ReactElement, useState } from "react";

import { Align, Button, Form, Synnax, Text } from "@synnaxlabs/pluto";
import { UseMutationResult, useMutation } from "@tanstack/react-query";

import { GroupConfig } from "@/hardware/ni/device/types";

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
