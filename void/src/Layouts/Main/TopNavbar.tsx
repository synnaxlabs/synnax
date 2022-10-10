import { Navbar } from "@synnaxlabs/pluto";

export default function TopNavbar() {
  return (
    <Navbar data-tauri-drag-region location="top" size={36}>
      <Navbar.Start></Navbar.Start>
      <Navbar.Center />
      <Navbar.End />
    </Navbar>
  );
}
