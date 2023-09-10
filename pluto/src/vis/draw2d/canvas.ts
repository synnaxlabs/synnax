// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scale } from "@synnaxlabs/x";

export class SugaredOffscreenCanvasRenderingContext2D
  implements OffscreenCanvasRenderingContext2D
{
  readonly scale_: scale.XY;
  private readonly wrap: OffscreenCanvasRenderingContext2D;

  constructor(
    wrap: OffscreenCanvasRenderingContext2D,
    scale_: scale.XY = scale.XY.IDENTITY,
  ) {
    this.wrap = wrap;
    this.scale_ = scale_;
  }

  reset(): void {
    this.wrap.reset();
  }

  applyScale(scale: scale.XY): SugaredOffscreenCanvasRenderingContext2D {
    return new SugaredOffscreenCanvasRenderingContext2D(this.wrap, scale);
  }

  get canvas(): OffscreenCanvas {
    return this.wrap.canvas;
  }

  get miterLimit(): number {
    return this.wrap.miterLimit;
  }

  set miterLimit(value: number) {
    this.wrap.miterLimit = this.scale_.x.dim(value);
  }

  get globalAlpha(): number {
    return this.wrap.globalAlpha;
  }

  set globalAlpha(value: number) {
    this.wrap.globalAlpha = value;
  }

  get globalCompositeOperation(): GlobalCompositeOperation {
    return this.wrap.globalCompositeOperation;
  }

  set globalCompositeOperation(value: GlobalCompositeOperation) {
    this.wrap.globalCompositeOperation = value;
  }

  get fillStyle(): string | CanvasGradient | CanvasPattern {
    return this.wrap.fillStyle;
  }

  set fillStyle(value: string | CanvasGradient | CanvasPattern) {
    this.wrap.fillStyle = value;
  }

  get strokeStyle(): string | CanvasGradient | CanvasPattern {
    return this.wrap.strokeStyle;
  }

  set strokeStyle(value: string | CanvasGradient | CanvasPattern) {
    this.wrap.strokeStyle = value;
  }

  commit(): void {
    this.wrap.commit();
  }

  drawImage(image: CanvasImageSource, dx: number, dy: number): void;
  drawImage(
    image: CanvasImageSource,
    dx: number,
    dy: number,
    dw: number,
    dh: number,
  ): void;
  drawImage(
    image: CanvasImageSource,
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    dx: number,
    dy: number,
    dw: number,
    dh: number,
  ): void;
  drawImage(image: CanvasImageSource, dx: number, dy: number): void;
  drawImage(
    image: CanvasImageSource,
    dx: number,
    dy: number,
    dw: number,
    dh: number,
  ): void;
  drawImage(
    image: CanvasImageSource,
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    dx: number,
    dy: number,
    dw: number,
    dh: number,
  ): void;
  drawImage(
    image: unknown,
    sx: unknown,
    sy: unknown,
    sw?: unknown,
    sh?: unknown,
    dx?: unknown,
    dy?: unknown,
    dw?: unknown,
    dh?: unknown,
  ): void {
    // @ts-expect-error - typescript overloads cause issues here
    this.wrap.drawImage(image, sx, sy, sw, sh, dx, dy, dw, dh);
  }

  beginPath(): void {
    this.wrap.beginPath();
  }

  clip(fillRule?: CanvasFillRule | undefined): void;
  clip(path: Path2D, fillRule?: CanvasFillRule | undefined): void;
  clip(path?: unknown, fillRule?: unknown): void {
    // @ts-expect-error - typescript overloads cause issues here
    this.wrap.clip(path, fillRule);
  }

  fill(fillRule?: CanvasFillRule | undefined): void;
  fill(path: Path2D, fillRule?: CanvasFillRule | undefined): void;
  fill(path?: unknown, fillRule?: unknown): void {
    if (path == null) return this.wrap.fill();
    // @ts-expect-error - typescript overloads cause issues here
    this.wrap.fill(path, fillRule);
  }

  isPointInPath(x: number, y: number, fillRule?: CanvasFillRule | undefined): boolean;
  isPointInPath(
    path: Path2D,
    x: number,
    y: number,
    fillRule?: CanvasFillRule | undefined,
  ): boolean;
  isPointInPath(path: unknown, x: unknown, y?: unknown, fillRule?: unknown): boolean {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrap.isPointInPath(path, x, y, fillRule);
  }

  isPointInStroke(x: number, y: number): boolean;
  isPointInStroke(path: Path2D, x: number, y: number): boolean;
  isPointInStroke(path: unknown, x: unknown, y?: unknown): boolean {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrap.isPointInStroke(path, x, y);
  }

  stroke(): void;
  stroke(path: Path2D): void;
  stroke(path?: Path2D): void {
    if (path == null) return this.wrap.stroke();
    this.wrap.stroke(path);
  }

  createConicGradient(startAngle: number, x: number, y: number): CanvasGradient {
    return this.wrap.createConicGradient(startAngle, x, y);
  }

  createLinearGradient(x0: number, y0: number, x1: number, y1: number): CanvasGradient {
    return this.wrap.createLinearGradient(x0, y0, x1, y1);
  }

  createPattern(
    image: CanvasImageSource,
    repetition: string | null,
  ): CanvasPattern | null {
    return this.wrap.createPattern(image, repetition);
  }

  createRadialGradient(
    x0: number,
    y0: number,
    r0: number,
    x1: number,
    y1: number,
    r1: number,
  ): CanvasGradient {
    return this.wrap.createRadialGradient(x0, y0, r0, x1, y1, r1);
  }

  get filter(): string {
    return this.wrap.filter;
  }

  set filter(value: string) {
    this.wrap.filter = value;
  }

  createImageData(
    sw: number,
    sh: number,
    settings?: ImageDataSettings | undefined,
  ): ImageData;
  createImageData(imagedata: ImageData): ImageData;
  createImageData(sw: unknown, sh?: unknown, settings?: unknown): ImageData {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrap.createImageData(sw, sh, settings);
  }

  getImageData(
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    settings?: ImageDataSettings | undefined,
  ): ImageData {
    return this.wrap.getImageData(sx, sy, sw, sh, settings);
  }

  putImageData(imagedata: ImageData, dx: number, dy: number): void;
  putImageData(
    imagedata: ImageData,
    dx: number,
    dy: number,
    dirtyX: number,
    dirtyY: number,
    dirtyWidth: number,
    dirtyHeight: number,
  ): void;
  putImageData(
    imagedata: unknown,
    dx: unknown,
    dy: unknown,
    dirtyX?: unknown,
    dirtyY?: unknown,
    dirtyWidth?: unknown,
    dirtyHeight?: unknown,
  ): void {
    // @ts-expect-error - typescript overloads cause issues here
    this.wrap.putImageData(imagedata, dx, dy, dirtyX, dirtyY, dirtyWidth, dirtyHeight);
  }

  get imageSmoothingEnabled(): boolean {
    return this.wrap.imageSmoothingEnabled;
  }

  set imageSmoothingEnabled(value: boolean) {
    this.wrap.imageSmoothingEnabled = value;
  }

  get imageSmoothingQuality(): ImageSmoothingQuality {
    return this.wrap.imageSmoothingQuality;
  }

  set imageSmoothingQuality(value: ImageSmoothingQuality) {
    this.wrap.imageSmoothingQuality = value;
  }

  arc(
    x: number,
    y: number,
    radius: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean | undefined,
  ): void {
    this.wrap.arc(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radius),
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  arcTo(x1: number, y1: number, x2: number, y2: number, radius: number): void {
    this.wrap.arcTo(
      this.scale_.x.pos(x1),
      this.scale_.y.pos(y1),
      this.scale_.x.pos(x2),
      this.scale_.y.pos(y2),
      this.scale_.x.dim(radius),
    );
  }

  bezierCurveTo(
    cp1x: number,
    cp1y: number,
    cp2x: number,
    cp2y: number,
    x: number,
    y: number,
  ): void {
    this.wrap.bezierCurveTo(
      this.scale_.x.pos(cp1x),
      this.scale_.y.pos(cp1y),
      this.scale_.x.pos(cp2x),
      this.scale_.y.pos(cp2y),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  closePath(): void {
    this.wrap.closePath();
  }

  ellipse(
    x: number,
    y: number,
    radiusX: number,
    radiusY: number,
    rotation: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean | undefined,
  ): void {
    this.wrap.ellipse(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radiusX),
      this.scale_.y.dim(radiusY),
      rotation,
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  lineTo(x: number, y: number): void {
    this.wrap.lineTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  moveTo(x: number, y: number): void {
    this.wrap.moveTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  quadraticCurveTo(cpx: number, cpy: number, x: number, y: number): void {
    this.wrap.quadraticCurveTo(
      this.scale_.x.pos(cpx),
      this.scale_.y.pos(cpy),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  rect(x: number, y: number, w: number, h: number): void {
    this.wrap.rect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  roundRect(
    x: number,
    y: number,
    w: number,
    h: number,
    radii?: number | DOMPointInit | Array<number | DOMPointInit> | undefined,
  ): void {
    this.wrap.roundRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
      typeof radii === "number" ? this.scale_.x.dim(radii) : radii,
    );
  }

  get lineCap(): CanvasLineCap {
    return this.wrap.lineCap;
  }

  set lineCap(value: CanvasLineCap) {
    this.wrap.lineCap = value;
  }

  get lineDashOffset(): number {
    return this.wrap.lineDashOffset;
  }

  set lineDashOffset(value: number) {
    this.wrap.lineDashOffset = this.scale_.x.dim(value);
  }

  get lineJoin(): CanvasLineJoin {
    return this.wrap.lineJoin;
  }

  set lineJoin(value: CanvasLineJoin) {
    this.wrap.lineJoin = value;
  }

  get lineWidth(): number {
    return this.wrap.lineWidth;
  }

  set lineWidth(value: number) {
    this.wrap.lineWidth = this.scale_.x.dim(value);
  }

  getLineDash(): number[] {
    return this.wrap.getLineDash();
  }

  setLineDash(segments: number[]): void;
  setLineDash(segments: Iterable<number>): void;
  setLineDash(segments: number[] | Iterable<number>): void {
    const scaled = Array.from(segments).map((v) => this.scale_.x.dim(v));
    this.wrap.setLineDash(scaled);
  }

  clearRect(x: number, y: number, w: number, h: number): void {
    this.wrap.clearRect(x, y, w, h);
  }

  fillRect(x: number, y: number, w: number, h: number): void {
    this.wrap.fillRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  strokeRect(x: number, y: number, w: number, h: number): void {
    this.wrap.strokeRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  get shadowBlur(): number {
    return this.wrap.shadowBlur;
  }

  set shadowBlur(value: number) {
    this.wrap.shadowBlur = value;
  }

  get shadowColor(): string {
    return this.wrap.shadowColor;
  }

  set shadowColor(value: string) {
    this.wrap.shadowColor = value;
  }

  get shadowOffsetX(): number {
    return this.wrap.shadowOffsetX;
  }

  set shadowOffsetX(value: number) {
    this.wrap.shadowOffsetX = value;
  }

  get shadowOffsetY(): number {
    return this.wrap.shadowOffsetY;
  }

  restore(): void {
    this.wrap.restore();
  }

  save(): void {
    this.wrap.save();
  }

  fillText(text: string, x: number, y: number, maxWidth?: number | undefined): void {
    this.wrap.fillText(
      text,
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      maxWidth != null ? this.scale_.x.dim(maxWidth) : undefined,
    );
  }

  measureText(text: string): TextMetrics {
    this.wrap.font = scaleFontSize(this.wrap.font, this.scale_.x.reverse());
    const metrics = this.wrap.measureText(text);
    this.wrap.font = scaleFontSize(this.wrap.font, this.scale_.x);
    return metrics;
  }

  strokeText(text: string, x: number, y: number, maxWidth?: number | undefined): void {
    this.wrap.strokeText(
      text,
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      maxWidth != null ? this.scale_.x.dim(maxWidth) : undefined,
    );
  }

  get direction(): CanvasDirection {
    return this.wrap.direction;
  }

  set direction(value: CanvasDirection) {
    this.wrap.direction = value;
  }

  get font(): string {
    return this.wrap.font;
  }

  set font(value: string) {
    this.wrap.font = scaleFontSize(value, this.scale_.x);
  }

  get fontKerning(): CanvasFontKerning {
    return this.wrap.fontKerning;
  }

  set fontKerning(value: CanvasFontKerning) {
    this.wrap.fontKerning = value;
  }

  get textAlign(): CanvasTextAlign {
    return this.wrap.textAlign;
  }

  set textAlign(value: CanvasTextAlign) {
    this.wrap.textAlign = value;
  }

  get textBaseline(): CanvasTextBaseline {
    return this.wrap.textBaseline;
  }

  set textBaseline(value: CanvasTextBaseline) {
    this.wrap.textBaseline = value;
  }

  getTransform(): DOMMatrix {
    return this.wrap.getTransform();
  }

  resetTransform(): void {
    this.wrap.resetTransform();
  }

  rotate(angle: number): void {
    this.wrap.rotate(angle);
  }

  scale(x: number, y: number): void {
    this.wrap.scale(x, y);
  }

  setTransform(a: number, b: number, c: number, d: number, e: number, f: number): void;
  setTransform(transform?: DOMMatrix2DInit | undefined): void;
  setTransform(
    a?: unknown,
    b?: unknown,
    c?: unknown,
    d?: unknown,
    e?: unknown,
    f?: unknown,
  ): void {
    // @ts-expect-error
    this.wrap.setTransform(a, b, c, d, e, f);
  }

  transform(a: number, b: number, c: number, d: number, e: number, f: number): void {
    this.wrap.transform(a, b, c, d, e, f);
  }

  translate(x: number, y: number): void {
    this.wrap.translate(x, y);
  }
}

// fInd the term 'px' and then get all the numbers before it, INCLUDING DECIMALS.
// /(\d+)px/ is wrong, so don't use it.
const FONT_REGEX = /(\d+(\.\d+)?)px/;

const scaleFontSize = (font: string, scale: scale.Scale): string => {
  const fontSize = Number(font.match(FONT_REGEX)?.[1]);
  if (fontSize == null) return font;
  const scaled = scale.dim(fontSize);
  return font.replace(FONT_REGEX, `${scaled}px`);
};
