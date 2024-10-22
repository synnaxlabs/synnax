import { Align, Button } from "@synnaxlabs/pluto";
import { useState } from "react";

export const ModeSwitch = () => {
  const [active, setActive] = useState("community");
  return (
    <Align.Space direction="x" className="mode-switch" align="center" justify="center">
      <Button.Button
        size="small"
        variant="text"
        className={active === "community" ? "active" : ""}
      >
        Community
      </Button.Button>
      <Button.Button
        size="small"
        variant="text"
        className={active === "enterprise" ? "active" : ""}
      >
        Enterprise
      </Button.Button>
    </Align.Space>
  );
};
