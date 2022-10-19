import { Nav } from "@synnaxlabs/pluto";
import RangeSelectorItem from "../../range/RangeSelector";

export default function RightNavbar() {
  return <Nav.Drawer location="right" size={48} items={[RangeSelectorItem]} />;
}
