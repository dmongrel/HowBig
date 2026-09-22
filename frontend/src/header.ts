// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// The header above the map: one centered line per selected country, in its side's color.

import {areaLine} from "./format";

export interface HeaderLine {
    name: string;
    area: number;
    color: string;
}

/** Replaces host's lines with one per entry, left first. */
export function renderHeader(host: HTMLElement, lines: HeaderLine[]): void {
    host.replaceChildren(...lines.map((l) => {
        const div = document.createElement("div");
        div.className = "header-line";
        div.style.color = l.color;
        div.textContent = areaLine(l.name, l.area);
        return div;
    }));
}
