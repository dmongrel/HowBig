// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Global keyboard shortcuts. ESC works from anywhere, search boxes included;
// F and A work whenever focus is not in a text field. v1.0.0 needed a click
// on the map first.

export interface KeyActions {
    quit(): void;
    toggleFullscreen(): void;
    about(): void;
}

/** True when the event target takes typed text. */
export function inTextField(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) {
        return false;
    }
    if (target.isContentEditable || target instanceof HTMLTextAreaElement) {
        return true;
    }
    return target instanceof HTMLInputElement && !["button", "checkbox", "radio", "submit", "reset"].includes(target.type);
}

/** Installs the shortcuts on window. The capture phase runs before a dialog's own ESC handling. */
export function installKeys(actions: KeyActions): void {
    window.addEventListener("keydown", (e) => {
        if (e.key === "Escape") {
            e.preventDefault();
            actions.quit();
            return;
        }
        if (e.repeat || e.ctrlKey || e.altKey || e.metaKey || inTextField(e.target)) {
            return;
        }
        switch (e.key.toLowerCase()) {
            case "f":
                e.preventDefault();
                actions.toggleFullscreen();
                break;
            case "a":
                e.preventDefault();
                actions.about();
                break;
        }
    }, {capture: true});
}
