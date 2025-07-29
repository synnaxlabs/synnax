import { box, invert, position } from "@synnaxlabs/x";

export type Variant = "connected" | "floating" | "modal";

interface CalcDialogProps extends Pick<position.DialogProps, "initial" | "prefer"> {
  variant: Variant;
  target: HTMLElement;
  dialog: HTMLElement;
}

const FLOATING_PROPS: Partial<position.DialogProps> = {
  alignments: ["end"],
  disable: ["center"],
  prefer: [{ y: "bottom" }],
};
const FLOATING_TRANSLATE_AMOUNT: number = 3;

const positionFloatingDialog = ({
  target: target_,
  dialog: dialog_,
  initial,
  prefer,
}: CalcDialogProps): position.DialogReturn => {
  const res = position.dialog({
    container: box.construct(0, 0, window.innerWidth, window.innerHeight),
    target: box.construct(target_),
    dialog: box.construct(dialog_),
    ...FLOATING_PROPS,
    initial,
    prefer,
  });
  const { location } = res;
  const adjustedDialog = box.translate(
    res.adjustedDialog,
    "y",
    invert(location.y === "top") * FLOATING_TRANSLATE_AMOUNT,
  );
  return { adjustedDialog, location };
};

const CONNECTED_PROPS: Partial<position.DialogProps> = {
  alignments: ["center"],
  disable: [{ y: "center" }],
  initial: { x: "center" },
  prefer: [{ y: "bottom" }],
};
const CONNECTED_TRANSLATE_AMOUNT: number = -3;

const positionConnectedDialog = ({
  target,
  dialog,
  initial = CONNECTED_PROPS.initial,
  prefer = CONNECTED_PROPS.prefer,
}: CalcDialogProps): position.DialogReturn => {
  const targetBox = box.construct(target);
  const props: position.DialogProps = {
    target: targetBox,
    dialog: box.resize(box.construct(dialog), "x", box.width(targetBox)),
    container: box.construct(0, 0, window.innerWidth, window.innerHeight),
    ...CONNECTED_PROPS,
    initial,
    prefer,
  };
  const res = position.dialog(props);
  const { location } = res;
  const adjustedDialog = box.translate(
    res.adjustedDialog,
    "y",
    invert(location.y === "bottom") * CONNECTED_TRANSLATE_AMOUNT,
  );
  return { adjustedDialog, location };
};

export const positionDialog = (props: CalcDialogProps): position.DialogReturn => {
  if (props.variant === "connected") return positionConnectedDialog(props);
  return positionFloatingDialog(props);
};
