// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Modal message dialogs with an OK button, standing in for Fyne's dialog.NewCustom.
// Each call opens its own <dialog>, so a later one stacks on top of an earlier one.

/** Opens a modal dialog and resolves when it closes. */
export function showMessage(title: string, message: string): Promise<void> {
    const dialog = document.createElement("dialog");
    dialog.className = "message";

    const heading = document.createElement("h2");
    heading.textContent = title;

    const body = document.createElement("div");
    body.className = "message-body";
    body.textContent = message;

    const actions = document.createElement("div");
    actions.className = "message-actions";
    const ok = document.createElement("button");
    ok.type = "button";
    ok.textContent = "OK";
    ok.addEventListener("click", () => dialog.close());
    actions.append(ok);

    dialog.append(heading, body, actions);
    document.body.append(dialog);

    return new Promise((resolve) => {
        dialog.addEventListener("close", () => {
            dialog.remove();
            resolve();
        }, {once: true});
        dialog.showModal();
        ok.focus();
    });
}

/** Shows an error dialog titled "Error". */
export function showError(message: string): Promise<void> {
    return showMessage("Error", message);
}

let aboutOpen = false;

/** Shows the About dialog unless it is already open. */
export function showAbout(version: string): void {
    if (aboutOpen) {
        return;
    }
    aboutOpen = true;
    const attribution = "geoBoundaries data is used under CC-BY 4.0 license.\nFor more information refer to ATTRIBUTION.md";
    const shortcuts = "ESC - Exits Program.\nF - Toggles Fullscreen.\nA - Shows About information.";
    const msg = `HowBig ${version} Copyright © Joel L. Caesar.\n\n${attribution}\n\n${shortcuts}`;
    void showMessage("About", msg).then(() => {
        aboutOpen = false;
    });
}
