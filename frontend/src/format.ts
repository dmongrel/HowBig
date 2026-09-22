// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Pure helpers shared by the header, bars and map.

/** Square miles to square kilometres, as in v1.0.0's updateHeader. */
export const sqMiToSqKm = 2.58998811;

/** Fyne drew fills and bars at A=127. */
export const fillAlpha = 127 / 255;

export interface RGB {
    r: number;
    g: number;
    b: number;
}

/**
 * Rounds to a whole number the way Go's %.0f does: halves go to the even neighbour.
 * JS's Math.round and toFixed round halves up, which differs at exact .5 values.
 */
export function roundHalfEven(n: number): number {
    const r = Math.round(n);
    if (r - n === 0.5 && r % 2 !== 0) {
        return r - 1;
    }
    return r;
}

/**
 * Formats n with comma thousands separators, rounded to a whole number.
 * Matches Go's formatNumber for non-negative values.
 */
export function formatNumber(n: number): string {
    const digits = roundHalfEven(n).toFixed(0);
    let out = "";
    for (let i = digits.length - 1, j = 0; i >= 0; i--, j++) {
        if (j > 0 && j % 3 === 0) {
            out = "," + out;
        }
        out = digits[i] + out;
    }
    return out;
}

/** The header line for one country: `Name: 1,234 sq. mi. / 3,196 km.` */
export function areaLine(name: string, areaMi: number): string {
    return `${name}: ${formatNumber(areaMi)} sq. mi. / ${formatNumber(areaMi * sqMiToSqKm)} km.`;
}

/**
 * Parses "#RRGGBB" like Go's ParseHexColor. Anything else is white.
 */
export function parseHexColor(s: string): RGB {
    const m = /^#([0-9a-fA-F]{2})([0-9a-fA-F]{2})([0-9a-fA-F]{2})$/.exec(s);
    if (!m) {
        return {r: 255, g: 255, b: 255};
    }
    return {r: parseInt(m[1], 16), g: parseInt(m[2], 16), b: parseInt(m[3], 16)};
}

/** A CSS rgba() string for c at alpha a (0..1). */
export function cssColor(c: RGB, a = 1): string {
    return `rgba(${c.r}, ${c.g}, ${c.b}, ${a})`;
}
