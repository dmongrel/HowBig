// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// @vitest-environment happy-dom

import {describe, expect, it} from "vitest";
import {inTextField} from "./keys";

function input(type: string): HTMLInputElement {
    const el = document.createElement("input");
    el.type = type;
    return el;
}

describe("inTextField", () => {
    it.each(["text", "search", "number", "email", "password"])("is true for an input of type %s", (type) => {
        expect(inTextField(input(type))).toBe(true);
    });

    it.each(["button", "checkbox", "radio", "submit", "reset"])("is false for an input of type %s", (type) => {
        expect(inTextField(input(type))).toBe(false);
    });

    it("is true for a textarea", () => {
        expect(inTextField(document.createElement("textarea"))).toBe(true);
    });

    it("is true for a contenteditable element", () => {
        const div = document.createElement("div");
        div.contentEditable = "true";
        document.body.append(div);
        expect(inTextField(div)).toBe(true);
        div.remove();
    });

    it("is false for other elements, non-elements and null", () => {
        expect(inTextField(document.createElement("div"))).toBe(false);
        expect(inTextField(document.createElement("button"))).toBe(false);
        expect(inTextField(window)).toBe(false);
        expect(inTextField(null)).toBe(false);
    });
});
