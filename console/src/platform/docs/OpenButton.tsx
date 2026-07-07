import { Button, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useOpenTab } from "@/platform/docs/useOpenTab";

export const OpenButton = (): ReactElement => {
  const handleOpen = useOpenTab();
  return (
    <Button.Button
      size="small"
      variant="text"
      onClick={handleOpen}
      contrast={2}
      className={CSS.BE("docs", "open-button")}
      tooltip="Open Documentation"
    >
      <Icon.QuestionMark />
    </Button.Button>
  );
};
