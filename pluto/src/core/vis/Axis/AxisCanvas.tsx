import { locToDir } from "@synnaxlabs/x";

import { AxisContext, AxisProps } from "./core";

import { TickFactory, newTickFactory } from "@/core/vis/Axis/TickFactory";
import { RenderContext } from "@/core/vis/render";

export class AxisCanvas {
  ctx: RenderContext;
  props: AxisProps;
  tickFactory: TickFactory;

  constructor(ctx: RenderContext, props: AxisProps) {
    this.ctx = ctx;
    this.props = props;
    this.tickFactory = newTickFactory(this.props);
  }

  setProps(props: AxisProps): void {
    this.props = props;
    this.tickFactory = newTickFactory(props);
  }

  render(ctx: AxisContext): void {
    const { region, position: _position, size } = ctx;
    const { location, showGrid = false, gridSize = 0 } = this.props;
    const dir = locToDir(location);
    const ticks = this.tickFactory.generate(ctx);
    const { canvas } = this.ctx;
    const pos = {
      x: region.x + _position.x,
      y: region.y + _position.y,
    };

    // Set some important canvas properties.
    canvas.strokeStyle = this.props.color;
    canvas.font = this.props.tickFont;

    // Start off by drawing the axis line.
    canvas.beginPath();
    canvas.moveTo(pos.x, pos.y);
    if (dir === "x") canvas.lineTo(pos.x + size, pos.y);
    else canvas.lineTo(pos.x, pos.y + size);
    canvas.stroke();

    // Draw the ticks
    ticks.forEach((tick) => {
      if (dir === "x") {
        canvas.moveTo(pos.x + tick.position, pos.y);
        if (location === "top") {
          canvas.lineTo(pos.x + tick.position, pos.y - 5);
          canvas.fillText(tick.label, pos.x + tick.position, pos.y - 10);
          if (showGrid) {
            canvas.beginPath();
            canvas.moveTo(pos.x + tick.position, pos.y);
            canvas.lineTo(pos.x + tick.position, pos.y - gridSize);
            canvas.stroke();
          }
        } else {
          canvas.lineTo(pos.x + tick.position, pos.y + 5);
          canvas.fillText(tick.label, pos.x + tick.position, pos.y + 10);
          if (showGrid) {
            canvas.beginPath();
            canvas.moveTo(pos.x + tick.position, pos.y);
            canvas.lineTo(pos.x + tick.position, pos.y + gridSize);
            canvas.stroke();
          }
        }
      } else {
        canvas.moveTo(pos.x, pos.y + tick.position);
        if (location === "left") {
          canvas.lineTo(pos.x - 5, pos.y + tick.position);
          canvas.fillText(tick.label, pos.x - 10, pos.y + tick.position);
          if (showGrid) {
            canvas.beginPath();
            canvas.moveTo(pos.x, pos.y + tick.position);
            canvas.lineTo(pos.x - gridSize, pos.y + tick.position);
            canvas.stroke();
          }
        } else {
          canvas.lineTo(pos.x + 5, pos.y + tick.position);
          canvas.fillText(tick.label, pos.x + 10, pos.y + tick.position);
          if (showGrid) {
            canvas.beginPath();
            canvas.moveTo(pos.x, pos.y + tick.position);
            canvas.lineTo(pos.x + gridSize, pos.y + tick.position);
            canvas.stroke();
          }
        }
      }
    });
  }
}
