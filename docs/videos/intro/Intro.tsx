import { Logo } from "@synnaxlabs/media";
import "@synnaxlabs/pluto/dist/pluto.css";
import "@synnaxlabs/media/dist/media.css";

import "./Intro.css";

const TRANSLATION = "translate(-50%, -50%)";

export const MyComposition = () => {
  return (
    <div style={{ width: "100%", height: "100%", backgroundColor: "black" }}>
      <div className="logos">
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
      </div>
    </div>
  );
};
