import { NavDrawer } from "@synnaxlabs/pluto";
import RangeSelectorItem from "../../range/RangeSelector";

export default function RightNavbar() {
  return <NavDrawer location="right" size={48} items={[RangeSelectorItem]} />;
}
