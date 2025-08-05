import { type Button } from "@/button";
import { type Dialog } from "@/dialog";

export type Variant = Dialog.FrameProps["variant"] | "preview";

export const transformDialogVariant = (
  variant: Variant,
): Dialog.FrameProps["variant"] => (variant === "preview" ? "connected" : variant);

export const transformTriggerVariant = (
  variant: Variant,
): Button.ButtonProps["variant"] => (variant === "preview" ? "preview" : undefined);
