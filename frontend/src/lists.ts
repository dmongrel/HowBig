// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// A searchable country list: search box with an "X" clear button, the rows,
// and a "Deselect All" button, as in v1.0.0's createList.

/** Horizontal padding inside a row, each side (Fyne's container.NewPadded at padding 4). */
export const rowPaddingX = 4;

/** Scrollbar width, as in v1.0.0's customTheme. */
export const scrollbarWidth = 20;

/**
 * Builds a list into host. onChange gets the selected name, or "" when the
 * selection is cleared.
 */
export function createCountryList(host: HTMLElement, names: string[], onChange: (name: string) => void): void {
    host.replaceChildren();

    const search = document.createElement("div");
    search.className = "search";
    const input = document.createElement("input");
    input.type = "text";
    input.placeholder = "Search...";
    input.spellcheck = false;
    input.autocomplete = "off";
    const clear = document.createElement("button");
    clear.type = "button";
    clear.className = "clear";
    clear.textContent = "X";
    search.append(input, clear);

    const rowsBox = document.createElement("div");
    rowsBox.className = "rows";
    const rows = names.map((name) => {
        const row = document.createElement("div");
        row.className = "row";
        row.textContent = name;
        row.dataset.name = name;
        rowsBox.append(row);
        return row;
    });

    const deselectBtn = document.createElement("button");
    deselectBtn.type = "button";
    deselectBtn.className = "deselect";
    deselectBtn.textContent = "Deselect All";

    host.append(search, rowsBox, deselectBtn);

    let selected = "";
    const select = (name: string) => {
        if (name === selected) {
            return;
        }
        selected = name;
        for (const row of rows) {
            row.classList.toggle("selected", row.dataset.name === selected);
        }
        onChange(selected);
    };

    rowsBox.addEventListener("click", (e) => {
        const row = (e.target as HTMLElement).closest<HTMLElement>(".row");
        if (!row || row.dataset.name === undefined) {
            return;
        }
        // Clicking the selected row again deselects it, as in v1.0.0.
        select(row.dataset.name === selected ? "" : row.dataset.name);
    });

    const filter = () => {
        const term = input.value.toLowerCase();
        for (const row of rows) {
            row.hidden = !(row.dataset.name ?? "").toLowerCase().includes(term);
        }
    };
    input.addEventListener("input", filter);

    clear.addEventListener("click", () => {
        input.value = "";
        filter();
    });

    deselectBtn.addEventListener("click", () => select(""));
}

/** The width in CSS px of the widest name at the given font. */
export function widestName(names: string[], font: string): number {
    const ctx = document.createElement("canvas").getContext("2d");
    if (!ctx) {
        return 0;
    }
    ctx.font = font;
    let widest = 0;
    for (const name of names) {
        widest = Math.max(widest, ctx.measureText(name).width);
    }
    return widest;
}
