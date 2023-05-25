import { BoxT, ONE_DIMS, Dimensions, XY, ZERO_XY, Box, Deep } from "@synnaxlabs/x";

export interface ViewportState {
  zoom: Dimensions;
  pan: XY;
  viewport: BoxT;
}

const ZERO_VIEWPORT_STATE: ViewportState = {
  zoom: ONE_DIMS,
  pan: ZERO_XY,
  viewport: Box.ZERO,
};

export class Viewport {
  private state: ViewportState;

  static zeroState(): ViewportState {
    return Deep.copy(ZERO_VIEWPORT_STATE);
  }

  constructor() {
    this.state = Deep.copy(ZERO_VIEWPORT_STATE);
  }

  update(state: ViewportState): void {
    this.state = state;
  }

  get box(): Box {
    return new Box(this.state.viewport);
  }
}
