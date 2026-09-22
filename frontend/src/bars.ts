// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// The area bars beside the map, as in v1.0.0's drawBar: the fill is
// area/maxArea of the bar's height less 2px padding top and bottom, and at
// least 1px when the area is above zero. CSS resolves the height, so the bar
// follows resizes without a redraw.

/** Padding around the fill, as in drawBar. */
const padding = 2;

/**
 * Draws a bar for a selected country into host, or empties it when selected
 * is false. color is the fill as a CSS color.
 */
export function renderBar(host: HTMLElement, selected: boolean, area: number, maxArea: number, color: string): void {
    host.replaceChildren();
    if (!selected) {
        return;
    }
    const proportion = maxArea > 0 ? Math.min(area / maxArea, 1) : 0;
    const fill = document.createElement("div");
    fill.className = "bar-fill";
    fill.style.background = color;
    fill.style.left = `${padding}px`;
    fill.style.right = `${padding}px`;
    fill.style.bottom = `${padding}px`;
    const height = `calc((100% - ${padding * 2}px) * ${proportion})`;
    fill.style.height = area > 0 ? `max(1px, ${height})` : height;
    host.append(fill);
}
