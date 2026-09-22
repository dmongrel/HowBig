// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Draws a MapLayout as SVG, in v1.0.0's order: fill the larger country, fill
// the smaller one, then stroke the left border and the right border. Each
// polygon is its own <path>, as each was its own raster in v1.0.0, so
// overlapping 50% fills double up the same way.

import type {CountryLayout, MapLayout, Rect} from "../bindings/HowBig";

const svgNS = "http://www.w3.org/2000/svg";

export interface SideColors {
    fill: string;
    border: string;
}

/** Turns a flat [x0,y0,x1,y1,...] ring into SVG path data. */
function pathData(flat: number[]): string {
    let d = "";
    for (let i = 0; i + 1 < flat.length; i += 2) {
        d += `${i === 0 ? "M" : "L"}${flat[i]} ${flat[i + 1]}`;
    }
    return d + "Z";
}

function boxElement(box: Rect): SVGRectElement {
    const r = document.createElementNS(svgNS, "rect");
    r.setAttribute("x", String(box.x));
    r.setAttribute("y", String(box.y));
    r.setAttribute("width", String(box.width));
    r.setAttribute("height", String(box.height));
    r.setAttribute("class", "debug-box");
    return r;
}

/**
 * Appends one drawCountry pass for c: its debug box (when present), then one
 * path per ring, filled or stroked.
 */
function drawPass(g: SVGGElement, c: CountryLayout, fill: string | null, stroke: string | null): void {
    if (c.box) {
        g.append(boxElement(c.box));
    }
    for (const ring of c.paths ?? []) {
        if (!ring || ring.length < 6) {
            continue;
        }
        const p = document.createElementNS(svgNS, "path");
        p.setAttribute("d", pathData(ring));
        p.setAttribute("fill-rule", "evenodd");
        p.setAttribute("fill", fill ?? "none");
        if (stroke) {
            p.setAttribute("stroke", stroke);
            p.setAttribute("stroke-width", "1");
        }
        g.append(p);
    }
}

/** Replaces svg's contents with layout, sized width x height. */
export function renderMap(svg: SVGSVGElement, layout: MapLayout | null, width: number, height: number, colors: Record<string, SideColors>): void {
    svg.setAttribute("width", String(width));
    svg.setAttribute("height", String(height));
    svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
    const g = document.createElementNS(svgNS, "g");
    const countries = (layout?.countries ?? []).filter((c) => !c.error);

    const byOrder = [...countries].sort((a, b) => a.order - b.order);
    for (const c of byOrder) {
        drawPass(g, c, colors[c.side]?.fill ?? null, null);
    }
    for (const side of ["left", "right"]) {
        for (const c of countries.filter((c) => c.side === side)) {
            drawPass(g, c, null, colors[side]?.border ?? null);
        }
    }
    svg.replaceChildren(g);
}
