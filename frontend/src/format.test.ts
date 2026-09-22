// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

import {describe, expect, it} from "vitest";
import {areaLine, formatNumber, parseHexColor, roundHalfEven} from "./format";

describe("roundHalfEven", () => {
    it.each([
        [0.5, 0], [1.5, 2], [2.5, 2], [3.5, 4], [1234.5, 1234],
        [0.4, 0], [0.6, 1], [2.4999, 2], [7, 7],
    ])("rounds %d to %d like Go's %%.0f", (n, want) => {
        expect(roundHalfEven(n)).toBe(want);
    });
});

describe("formatNumber", () => {
    it.each([
        [0, "0"], [999, "999"], [1000, "1,000"], [12345, "12,345"],
        [1234567, "1,234,567"], [6601668, "6,601,668"],
        [999.5, "1,000"], [1000.5, "1,000"], [2.5, "2"],
    ])("formats %d as %s", (n, want) => {
        expect(formatNumber(n)).toBe(want);
    });

    it("builds the header line in square miles and kilometres", () => {
        expect(areaLine("Fiji", 7056)).toBe("Fiji: 7,056 sq. mi. / 18,275 km.");
    });
});

describe("parseHexColor", () => {
    it("parses #RRGGBB in either case", () => {
        expect(parseHexColor("#00FF00")).toEqual({r: 0, g: 255, b: 0});
        expect(parseHexColor("#ffcc00")).toEqual({r: 255, g: 204, b: 0});
        expect(parseHexColor("#1a2B3c")).toEqual({r: 0x1a, g: 0x2b, b: 0x3c});
    });

    it.each(["", "00FF00", "#0F0", "#00FF00FF", "#GG0000", " #00FF00", "red"])(
        "returns white for %j",
        (s) => {
            expect(parseHexColor(s)).toEqual({r: 255, g: 255, b: 255});
        },
    );
});
