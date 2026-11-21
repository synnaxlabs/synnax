#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the path of a compound icon: a base icon with a modifier at its top right.

Usage: uv run --project scripts scripts/compound_icon.py <base> <modifier>
    [--cut box|circle|outline] [--width <units>] [--line-width <units>]

<base> and <modifier> are SVG path data on the 24-unit icon grid. Give the result to
Icon.createFromPath. The rules are in docs/claude/ux.md under "Compound icons".
"""

import argparse
import math
from typing import Literal

import pathops
from fontTools.pens.basePen import decomposeQuadraticSegment
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.svgLib.path import parse_path

type Cut = Literal["box", "circle", "outline"]

MODIFIER_WIDTH = 9.0
# The top-right corner of the 20-unit live area of the grid.
MODIFIER_CORNER = (22.0, 2.0)
BASE_SCALE = 0.9
# The bottom-left corner of the 20-unit live area of the grid.
BASE_ORIGIN = (2.0, 22.0)
CLEARANCE = 1.75
CORNER_RADIUS = 0.5
# An opening as wide as a line would erase it, so a cap is a bit narrower.
CAP_FRACTION = 0.95
MIN_AREA = 0.5
# A piece the opening cuts off only touches the rounded zone, so look past its edge.
EDGE_TOLERANCE = 0.05
CURVE_SAMPLES = 16


def compound(
    base: str,
    modifier: str,
    cut: Cut = "box",
    width: float = MODIFIER_WIDTH,
    line_width: float | None = None,
) -> str:
    """Returns the path data of base with modifier at its top right.

    The base scales from the bottom-left corner of the live area. The cut clears the
    modifier on all sides, and the corners it makes are rounded.

    :param base: The SVG path data of the base icon.
    :param modifier: The SVG path data of the modifier.
    :param cut: The shape that clears the base around the modifier.
    :param width: The width of the modifier, centered in a square as wide at the
        top-right corner of the live area.
    :param line_width: The width of the lines of a line base. The lines the cut ends
        get round caps.
    :raises ValueError: If width or line_width is not positive, if modifier has no
        width, or if the path operations fail and the cut grows the base.
    """
    if width <= 0:
        raise ValueError("width must be positive")
    radius = CORNER_RADIUS
    if line_width is not None:
        if line_width <= 0:
            raise ValueError("line width must be positive")
        radius = BASE_SCALE * line_width / 2 * CAP_FRACTION
    b = _parse(base).transform(
        BASE_SCALE,
        0,
        0,
        BASE_SCALE,
        BASE_ORIGIN[0] * (1 - BASE_SCALE),
        BASE_ORIGIN[1] * (1 - BASE_SCALE),
    )
    center = (MODIFIER_CORNER[0] - width / 2, MODIFIER_CORNER[1] + width / 2)
    m = _place_modifier(_parse(modifier), width, center)
    region = _cut_region(m, cut, center)
    near = _grow(region, 2 * radius)
    cut_base = pathops.op(b, region, pathops.PathOp.DIFFERENCE)
    b = _round_cut_corners(cut_base, near, radius)
    # Skia path operations can fail on near-coincident edges.
    if b.area > cut_base.area + 1e-3:
        raise ValueError("rounding the cut grew the base: try another cut or width")
    return _format(pathops.op(_without_slivers(b, near), m, pathops.PathOp.UNION))


def _parse(d: str) -> pathops.Path:
    path = pathops.Path()
    parse_path(d, path.getPen())
    return pathops.simplify(path)


def _place_modifier(
    m: pathops.Path, width: float, center: tuple[float, float]
) -> pathops.Path:
    x_min, y_min, x_max, y_max = m.bounds
    if x_max <= x_min:
        raise ValueError("modifier has no width")
    scale = width / (x_max - x_min)
    return m.transform(
        scale,
        0,
        0,
        scale,
        center[0] - scale * (x_min + x_max) / 2,
        center[1] - scale * (y_min + y_max) / 2,
    )


def _cut_region(m: pathops.Path, cut: Cut, center: tuple[float, float]) -> pathops.Path:
    region = pathops.Path()
    if cut == "outline":
        return _grow(m, CLEARANCE)
    if cut == "circle":
        cx, cy = center
        r = CLEARANCE + max(math.dist(p, center) for p in _points(m))
        # A circle from four arcs of 90 degrees.
        k = 0.5523 * r
        region.moveTo(cx + r, cy)
        region.cubicTo(cx + r, cy + k, cx + k, cy + r, cx, cy + r)
        region.cubicTo(cx - k, cy + r, cx - r, cy + k, cx - r, cy)
        region.cubicTo(cx - r, cy - k, cx - k, cy - r, cx, cy - r)
        region.cubicTo(cx + k, cy - r, cx + r, cy - k, cx + r, cy)
        region.close()
        return region
    x_min, y_min, x_max, y_max = m.bounds
    region.moveTo(x_min - CLEARANCE, y_min - CLEARANCE)
    region.lineTo(x_max + CLEARANCE, y_min - CLEARANCE)
    region.lineTo(x_max + CLEARANCE, y_max + CLEARANCE)
    region.lineTo(x_min - CLEARANCE, y_max + CLEARANCE)
    region.close()
    return region


def _band(path: pathops.Path, width: float) -> pathops.Path:
    band = pathops.Path(path)
    band.stroke(width, pathops.LineCap.ROUND_CAP, pathops.LineJoin.ROUND_JOIN, 4)
    band.convertConicsToQuads()
    return pathops.simplify(band)


def _grow(path: pathops.Path, by: float) -> pathops.Path:
    return pathops.op(path, _band(path, 2 * by), pathops.PathOp.UNION)


def _shrink(path: pathops.Path, by: float) -> pathops.Path:
    return pathops.op(path, _band(path, 2 * by), pathops.PathOp.DIFFERENCE)


def _round_cut_corners(
    path: pathops.Path, near: pathops.Path, radius: float
) -> pathops.Path:
    # Shrinking then growing rounds every convex corner. Use that shape only near the
    # cut, so the corners the base icon draws stay as they are.
    opened = _grow(_shrink(path, radius), radius)
    return pathops.op(
        pathops.op(path, near, pathops.PathOp.DIFFERENCE),
        pathops.op(opened, near, pathops.PathOp.INTERSECTION),
        pathops.PathOp.UNION,
    )


def _without_slivers(path: pathops.Path, near: pathops.Path) -> pathops.Path:
    zone = _grow(near, EDGE_TOLERANCE)
    kept = pathops.Path()
    for contour in path.contours:
        cut = pathops.op(contour, zone, pathops.PathOp.INTERSECTION).area > 0
        if abs(contour.area) >= MIN_AREA or not cut:
            kept.addPath(contour)
    return pathops.simplify(kept)


class _PointPen:
    def __init__(self) -> None:
        self.points: list[tuple[float, float]] = []

    def moveTo(self, p: tuple[float, float]) -> None:
        self.points.append(p)

    def lineTo(self, p: tuple[float, float]) -> None:
        self.points.append(p)

    def curveTo(self, *ps: tuple[float, float]) -> None:
        p0, (p1, p2, p3) = self.points[-1], ps
        for i in range(1, CURVE_SAMPLES + 1):
            t = i / CURVE_SAMPLES
            u = 1 - t
            self.points.append(
                (
                    u**3 * p0[0]
                    + 3 * u * u * t * p1[0]
                    + 3 * u * t * t * p2[0]
                    + t**3 * p3[0],
                    u**3 * p0[1]
                    + 3 * u * u * t * p1[1]
                    + 3 * u * t * t * p2[1]
                    + t**3 * p3[1],
                )
            )

    def qCurveTo(self, *ps: tuple[float, float]) -> None:
        for control, end in decomposeQuadraticSegment(ps):
            p0 = self.points[-1]
            # A quadratic curve as the cubic curve with the same shape.
            self.curveTo(
                (
                    p0[0] + 2 * (control[0] - p0[0]) / 3,
                    p0[1] + 2 * (control[1] - p0[1]) / 3,
                ),
                (
                    end[0] + 2 * (control[0] - end[0]) / 3,
                    end[1] + 2 * (control[1] - end[1]) / 3,
                ),
                end,
            )

    def closePath(self) -> None:
        pass

    def endPath(self) -> None:
        pass


def _points(path: pathops.Path) -> list[tuple[float, float]]:
    pen = _PointPen()
    path.draw(pen)
    return pen.points


def _format(path: pathops.Path) -> str:
    pen = SVGPathPen(None, ntos=lambda v: f"{round(v, 2):g}")
    path.draw(pen)
    return str(pen.getCommands())


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("base", help="the base icon's SVG path data")
    parser.add_argument("modifier", help="the modifier's SVG path data")
    parser.add_argument(
        "--cut",
        choices=["box", "circle", "outline"],
        default="box",
        help="the shape that clears the base around the modifier",
    )
    parser.add_argument(
        "--width",
        type=float,
        default=MODIFIER_WIDTH,
        help="the width of the modifier, for one much wider than it is tall",
    )
    parser.add_argument(
        "--line-width",
        type=float,
        help="the width of the lines of a line base, to give its cut ends round caps",
    )
    args = parser.parse_args()
    try:
        print(compound(args.base, args.modifier, args.cut, args.width, args.line_width))
    except ValueError as e:
        parser.error(str(e))


if __name__ == "__main__":
    main()
