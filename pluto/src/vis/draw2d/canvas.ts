// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, Destructor, dimensions, scale, xy } from "@synnaxlabs/x";

import { applyOverScan } from "@/vis/render/util";

export class SugaredOffscreenCanvasRenderingContext2D
  implements OffscreenCanvasRenderingContext2D
{
  readonly scale_: scale.XY;
  readonly wrapped: OffscreenCanvasRenderingContext2D;

  constructor(
    wrap: OffscreenCanvasRenderingContext2D,
    scale_: scale.XY = scale.XY.IDENTITY,
  ) {
    this.wrapped = wrap;
    this.scale_ = scale_;
  }

  get fontStretch(): CanvasFontStretch {
    return this.wrapped.fontStretch;
  }

  set fontStretch(value: CanvasFontStretch) {
    this.wrapped.fontStretch = value;
  }

  get fontVariantCaps(): CanvasFontVariantCaps {
    return this.wrapped.fontVariantCaps;
  }

  set fontVariantCaps(value: CanvasFontVariantCaps) {
    this.wrapped.fontVariantCaps = value;
  }

  get wordSpacing(): string {
    return this.wrapped.wordSpacing;
  }

  set wordSpacing(value: string) {
    this.wrapped.wordSpacing = value;
  }

  get letterSpacing(): string {
    return this.wrapped.letterSpacing;
  }

  set letterSpacing(value: string) {
    this.wrapped.letterSpacing = value;
  }

  get textRendering(): CanvasTextRendering {
    return this.wrapped.textRendering;
  }

  set textRendering(value: CanvasTextRendering) {
    this.wrapped.textRendering = value;
  }

  reset(): void {
    this.wrapped.reset();
  }

  applyScale(scale: scale.XY): SugaredOffscreenCanvasRenderingContext2D {
    return new SugaredOffscreenCanvasRenderingContext2D(this, scale);
  }

  get canvas(): OffscreenCanvas {
    return this.wrapped.canvas;
  }

  get miterLimit(): number {
    return this.wrapped.miterLimit;
  }

  set miterLimit(value: number) {
    this.wrapped.miterLimit = this.scale_.x.dim(value);
  }

  get globalAlpha(): number {
    return this.wrapped.globalAlpha;
  }

  set globalAlpha(value: number) {
    this.wrapped.globalAlpha = value;
  }

  get globalCompositeOperation(): GlobalCompositeOperation {
    return this.wrapped.globalCompositeOperation;
  }

  set globalCompositeOperation(value: GlobalCompositeOperation) {
    this.wrapped.globalCompositeOperation = value;
  }

  get fillStyle(): string | CanvasGradient | CanvasPattern {
    return this.wrapped.fillStyle;
  }

  set fillStyle(value: string | CanvasGradient | CanvasPattern) {
    this.wrapped.fillStyle = value;
  }

  get strokeStyle(): string | CanvasGradient | CanvasPattern {
    return this.wrapped.strokeStyle;
  }

  set strokeStyle(value: string | CanvasGradient | CanvasPattern) {
    this.wrapped.strokeStyle = value;
  }

  commit(): void {
    this.wrapped.commit();
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
    this.wrapped.drawImage(image, sx, sy, sw, sh, dx, dy, dw, dh);
  }

  beginPath(): void {
    this.wrapped.beginPath();
  }

  clip(fillRule?: CanvasFillRule | undefined): void;
  clip(path: Path2D, fillRule?: CanvasFillRule | undefined): void;
  clip(path?: unknown, fillRule?: unknown): void {
    // @ts-expect-error - typescript overloads cause issues here
    this.wrapped.clip(path, fillRule);
  }

  fill(fillRule?: CanvasFillRule | undefined): void;
  fill(path: Path2D, fillRule?: CanvasFillRule | undefined): void;
  fill(path?: unknown, fillRule?: unknown): void {
    if (path == null) return this.wrapped.fill();
    // @ts-expect-error - typescript overloads cause issues here
    this.wrapped.fill(path, fillRule);
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
    return this.wrapped.isPointInPath(path, x, y, fillRule);
  }

  isPointInStroke(x: number, y: number): boolean;
  isPointInStroke(path: Path2D, x: number, y: number): boolean;
  isPointInStroke(path: unknown, x: unknown, y?: unknown): boolean {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrapped.isPointInStroke(path, x, y);
  }

  stroke(): void;
  stroke(path: Path2D): void;
  stroke(path?: Path2D): void {
    if (path == null) return this.wrapped.stroke();
    this.wrapped.stroke(path);
  }

  createConicGradient(startAngle: number, x: number, y: number): CanvasGradient {
    return this.wrapped.createConicGradient(startAngle, x, y);
  }

  createLinearGradient(x0: number, y0: number, x1: number, y1: number): CanvasGradient {
    return this.wrapped.createLinearGradient(x0, y0, x1, y1);
  }

  createPattern(
    image: CanvasImageSource,
    repetition: string | null,
  ): CanvasPattern | null {
    return this.wrapped.createPattern(image, repetition);
  }

  createRadialGradient(
    x0: number,
    y0: number,
    r0: number,
    x1: number,
    y1: number,
    r1: number,
  ): CanvasGradient {
    return this.wrapped.createRadialGradient(x0, y0, r0, x1, y1, r1);
  }

  get filter(): string {
    return this.wrapped.filter;
  }

  set filter(value: string) {
    this.wrapped.filter = value;
  }

  createImageData(
    sw: number,
    sh: number,
    settings?: ImageDataSettings | undefined,
  ): ImageData;
  createImageData(imagedata: ImageData): ImageData;
  createImageData(sw: unknown, sh?: unknown, settings?: unknown): ImageData {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrapped.createImageData(sw, sh, settings);
  }

  getImageData(
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    settings?: ImageDataSettings | undefined,
  ): ImageData {
    return this.wrapped.getImageData(sx, sy, sw, sh, settings);
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
    this.wrapped.putImageData(
      // @ts-expect-error - typescript overloads cause issues here
      imagedata,
      dx,
      dy,
      dirtyX,
      dirtyY,
      dirtyWidth,
      dirtyHeight,
    );
  }

  get imageSmoothingEnabled(): boolean {
    return this.wrapped.imageSmoothingEnabled;
  }

  set imageSmoothingEnabled(value: boolean) {
    this.wrapped.imageSmoothingEnabled = value;
  }

  get imageSmoothingQuality(): ImageSmoothingQuality {
    return this.wrapped.imageSmoothingQuality;
  }

  set imageSmoothingQuality(value: ImageSmoothingQuality) {
    this.wrapped.imageSmoothingQuality = value;
  }

  arc(
    x: number,
    y: number,
    radius: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean | undefined,
  ): void {
    this.wrapped.arc(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radius),
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  arcTo(x1: number, y1: number, x2: number, y2: number, radius: number): void {
    this.wrapped.arcTo(
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
    this.wrapped.bezierCurveTo(
      this.scale_.x.pos(cp1x),
      this.scale_.y.pos(cp1y),
      this.scale_.x.pos(cp2x),
      this.scale_.y.pos(cp2y),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  closePath(): void {
    this.wrapped.closePath();
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
    this.wrapped.ellipse(
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
    this.wrapped.lineTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  moveTo(x: number, y: number): void {
    this.wrapped.moveTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  quadraticCurveTo(cpx: number, cpy: number, x: number, y: number): void {
    this.wrapped.quadraticCurveTo(
      this.scale_.x.pos(cpx),
      this.scale_.y.pos(cpy),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  rect(x: number, y: number, w: number, h: number): void {
    this.wrapped.rect(
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
    this.wrapped.roundRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
      typeof radii === "number" ? this.scale_.x.dim(radii) : radii,
    );
  }

  get lineCap(): CanvasLineCap {
    return this.wrapped.lineCap;
  }

  set lineCap(value: CanvasLineCap) {
    this.wrapped.lineCap = value;
  }

  get lineDashOffset(): number {
    return this.wrapped.lineDashOffset;
  }

  set lineDashOffset(value: number) {
    this.wrapped.lineDashOffset = this.scale_.x.dim(value);
  }

  get lineJoin(): CanvasLineJoin {
    return this.wrapped.lineJoin;
  }

  set lineJoin(value: CanvasLineJoin) {
    this.wrapped.lineJoin = value;
  }

  get lineWidth(): number {
    return this.wrapped.lineWidth;
  }

  set lineWidth(value: number) {
    this.wrapped.lineWidth = this.scale_.x.dim(value);
  }

  getLineDash(): number[] {
    return this.wrapped.getLineDash();
  }

  setLineDash(segments: number[]): void;
  setLineDash(segments: Iterable<number>): void;
  setLineDash(segments: number[] | Iterable<number>): void {
    const scaled = Array.from(segments).map((v) => this.scale_.x.dim(v));
    this.wrapped.setLineDash(scaled);
  }

  clearRect(x: number, y: number, w: number, h: number): void {
    this.wrapped.clearRect(x, y, w, h);
  }

  fillRect(x: number, y: number, w: number, h: number): void {
    this.wrapped.fillRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  strokeRect(x: number, y: number, w: number, h: number): void {
    this.wrapped.strokeRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  get shadowBlur(): number {
    return this.wrapped.shadowBlur;
  }

  set shadowBlur(value: number) {
    this.wrapped.shadowBlur = value;
  }

  get shadowColor(): string {
    return this.wrapped.shadowColor;
  }

  set shadowColor(value: string) {
    this.wrapped.shadowColor = value;
  }

  get shadowOffsetX(): number {
    return this.wrapped.shadowOffsetX;
  }

  set shadowOffsetX(value: number) {
    this.wrapped.shadowOffsetX = value;
  }

  get shadowOffsetY(): number {
    return this.wrapped.shadowOffsetY;
  }

  restore(): void {
    this.wrapped.restore();
  }

  save(): void {
    this.wrapped.save();
  }

  fillText(text: string, x: number, y: number, maxWidth?: number | undefined): void {
    this.wrapped.fillText(
      text,
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      maxWidth != null ? this.scale_.x.dim(maxWidth) : undefined,
    );
  }

  measureText(text: string): TextMetrics {
    this.wrapped.font = scaleFontSize(this.wrapped.font, this.scale_.x.reverse());
    const metrics = this.wrapped.measureText(text);
    this.wrapped.font = scaleFontSize(this.wrapped.font, this.scale_.x);
    return metrics;
  }

  strokeText(text: string, x: number, y: number, maxWidth?: number | undefined): void {
    this.wrapped.strokeText(
      text,
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      maxWidth != null ? this.scale_.x.dim(maxWidth) : undefined,
    );
  }

  get direction(): CanvasDirection {
    return this.wrapped.direction;
  }

  set direction(value: CanvasDirection) {
    this.wrapped.direction = value;
  }

  get font(): string {
    return this.wrapped.font;
  }

  set font(value: string) {
    this.wrapped.font = scaleFontSize(value, this.scale_.x);
  }

  get fontKerning(): CanvasFontKerning {
    return this.wrapped.fontKerning;
  }

  set fontKerning(value: CanvasFontKerning) {
    this.wrapped.fontKerning = value;
  }

  get textAlign(): CanvasTextAlign {
    return this.wrapped.textAlign;
  }

  set textAlign(value: CanvasTextAlign) {
    this.wrapped.textAlign = value;
  }

  get textBaseline(): CanvasTextBaseline {
    return this.wrapped.textBaseline;
  }

  set textBaseline(value: CanvasTextBaseline) {
    this.wrapped.textBaseline = value;
  }

  getTransform(): DOMMatrix {
    return this.wrapped.getTransform();
  }

  resetTransform(): void {
    this.wrapped.resetTransform();
  }

  rotate(angle: number): void {
    this.wrapped.rotate(angle);
  }

  scale(x: number, y: number): void {
    this.wrapped.scale(x, y);
  }

  scissor(region: box.Box, overScan: xy.XY = xy.ZERO): Destructor {
    const p = new Path2D();
    region = applyOverScan(region, overScan);
    p.rect(...xy.couple(box.topLeft(region)), ...dimensions.couple(box.dims(region)));
    this.save();
    this.clip(p);
    return () => this.restore();
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
    // @ts-expect-error - canvas forwarding
    this.wrapped.setTransform(a, b, c, d, e, f);
  }

  transform(a: number, b: number, c: number, d: number, e: number, f: number): void {
    this.wrapped.transform(a, b, c, d, e, f);
  }

  translate(x: number, y: number): void {
    this.wrapped.translate(x, y);
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
