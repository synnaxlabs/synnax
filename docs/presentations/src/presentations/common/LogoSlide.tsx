import { Logo, Space } from "@synnaxlabs/pluto";
import { SlideContainer } from "../../components";
import CenteredSlide from "./CenteredSlide";

export default function LogoSlide() {
  return (
    <CenteredSlide>
      <Logo variant="title" style={{ width: "40%" }} />
    </CenteredSlide>
  );
}
