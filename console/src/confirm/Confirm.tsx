// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Align, Button, Nav, Status, Text, Triggers } from "@synnaxlabs/pluto";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { selectArgs } from "@/layout/selectors";
import { RootState } from "@/store";

interface ConfirmLayoutArgs {
  message: string;
  description: string;
  confirm?: {
    variant?: Status.Variant;
    label?: string;
  };
  cancel?: {
    variant?: Status.Variant;
    label?: string;
  };
  result?: boolean;
}

export const LAYOUT_TYPE = "confirm";

type LayoutOverrides = Omit<Partial<Layout.State>, "key" | "type">;

export const configureLayout = (
  args: ConfirmLayoutArgs,
  layoutOverrides?: LayoutOverrides,
): Layout.State<ConfirmLayoutArgs> => ({
  name: "Confirm",
  type: LAYOUT_TYPE,
  key: LAYOUT_TYPE,
  windowKey: LAYOUT_TYPE,
  location: "modal",
  window: {
    resizable: false,
    size: { height: 250, width: 700 },
    navTop: true,
  },
  ...layoutOverrides,
  args: {
    ...args,
    result: undefined,
  },
});

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Confirm: Layout.Renderer = ({ layoutKey, onClose }) => {
  const args = Layout.useSelectArgs<ConfirmLayoutArgs>(layoutKey);
  const { message, description, confirm, cancel } = args;
  const { variant: confirmVariant = "error", label: confirmLabel = "Confirm" } =
    confirm ?? {};
  const { variant: cancelVariant, label: cancelLabel = "Cancel" } = cancel ?? {};
  const dispatch = useDispatch();
  const handleResult = (value: boolean) => {
    dispatch(
      Layout.setArgs<ConfirmLayoutArgs>({
        key: layoutKey,
        args: { ...args, result: value },
      }),
    );
    onClose();
  };

  return (
    <Align.Space direction="y" className={CSS.B("confirm")} grow justify="center">
      <Align.Space
        direction="y"
        grow
        align="start"
        justify="center"
        style={{ padding: "5rem" }}
      >
        <Text.Text level="h3" shade={9} weight={450}>
          {message}
        </Text.Text>
        <Text.Text level="p" shade={7} weight={450}>
          {description}
        </Text.Text>
      </Align.Space>
      <Nav.Bar location="bottom" size="7rem">
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To {confirmLabel.toLowerCase()}
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End direction="x" align="center" style={{ paddingRight: "2rem" }}>
          <Button.Button
            variant="outlined"
            status={cancelVariant}
            onClick={() => handleResult(false)}
          >
            {cancelLabel}
          </Button.Button>
          <Button.Button
            status={confirmVariant}
            onClick={() => handleResult(true)}
            triggers={[SAVE_TRIGGER]}
          >
            {confirmLabel}
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

export type CreateConfirmModal = (
  args: ConfirmLayoutArgs,
  layoutOverrides?: LayoutOverrides,
) => Promise<boolean>;

export const useModal = (): CreateConfirmModal => {
  const placer = Layout.usePlacer();
  const store = useStore<RootState>();
  return async (args, layoutOverrides) => {
    let unsubscribe: ReturnType<typeof store.subscribe> | null = null;
    return await new Promise((resolve) => {
      const layout = configureLayout(args, layoutOverrides);
      placer(layout);
      unsubscribe = store.subscribe(() => {
        const l = Layout.select(store.getState(), layout.key);
        if (l == null) resolve(false);
        const args = selectArgs<ConfirmLayoutArgs>(store.getState(), layout.key);
        if (args.result == null) return;
        resolve(args.result);
        unsubscribe?.();
      });
    });
  };
};
