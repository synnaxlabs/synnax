import { Nav } from "@synnaxlabs/pluto";

export default function TopNavbar() {
  return (
    <Nav.Bar data-tauri-drag-region location="top" size={36}>
      <Nav.Bar.Start></Nav.Bar.Start>
    </Nav.Bar>
  );
}
