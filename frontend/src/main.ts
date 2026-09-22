// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Entry point: loads settings and countries from Go, builds the lists, and
// keeps the header, bars and map in step with the two selections.

import "./style.css";
import {CountryService, MapService, SettingsService, WindowService} from "../bindings/HowBig";
import type {MapLayout} from "../bindings/HowBig";
import type {Info as CountryInfo} from "../bindings/HowBig/internal/country";
import type {Settings} from "../bindings/HowBig/internal/settings";
import {renderBar} from "./bars";
import {showAbout, showError} from "./dialogs";
import {cssColor, fillAlpha, parseHexColor} from "./format";
import {renderHeader, type HeaderLine} from "./header";
import {installKeys} from "./keys";
import {LatestCall} from "./latest";
import {createCountryList, rowPaddingX, scrollbarWidth, widestName} from "./lists";
import {renderMap, type SideColors} from "./map";

/** Must match --font in style.css so list widths are measured in the font they draw in. */
const fontFamily = `"Noto Sans", "Segoe UI", system-ui, sans-serif`;

/** Panel border, both sides (style.css .panel). */
const panelBorder = 2 * 2;

/** Horizontal button padding, each side (style.css button). */
const buttonPaddingX = 8;

/** The debounce on map re-layout after a resize (R8). */
const resizeDebounceMs = 100;

function byId<T extends HTMLElement>(id: string): T {
    const el = document.getElementById(id);
    if (!el) {
        throw new Error(`missing #${id}`);
    }
    return el as T;
}

/** Pushes settings-driven sizes and colors into the CSS variables. */
function applySettings(s: Settings, names: string[]): void {
    const root = document.documentElement.style;
    root.setProperty("--bg", cssColor(parseHexColor(s.background_color)));
    root.setProperty("--button-font", `${s.button_font_size}px`);
    root.setProperty("--search-font", `${s.search_font_size}px`);
    root.setProperty("--list-font", `${s.country_list_font_size}px`);
    root.setProperty("--header-font", `${s.header_font_size}px`);

    // v1.0.0 sized the lists to the widest name plus 2px. Row padding and
    // the scrollbar are added here too so the longest name isn't clipped.
    // With no countries loaded, the "Deselect All" button sets the width instead.
    const widest = widestName(names, `${s.country_list_font_size}px ${fontFamily}`);
    const button = widestName(["Deselect All"], `${s.button_font_size}px ${fontFamily}`) + buttonPaddingX * 2;
    const width = Math.ceil(Math.max(widest + 2 + rowPaddingX * 2 + scrollbarWidth, button)) + panelBorder;
    root.setProperty("--list-width", `${width}px`);
}

async function start(): Promise<void> {
    // Buttons and shortcuts go in first so ESC still quits if loading fails.
    let version = "";
    const fullscreenBtn = byId<HTMLButtonElement>("fullscreen-btn");
    const quit = () => void WindowService.Quit();
    const toggleFullscreen = () => {
        WindowService.ToggleFullscreen()
            .then((full) => {
                fullscreenBtn.textContent = full ? "Windowed (F)" : "Fullscreen (F)";
            })
            .catch((err) => void showError(`Could not toggle fullscreen: ${err}`));
    };
    const about = () => showAbout(version);
    byId("exit-btn").addEventListener("click", quit);
    fullscreenBtn.addEventListener("click", toggleFullscreen);
    byId("about-btn").addEventListener("click", about);
    installKeys({quit, toggleFullscreen, about});

    let settings: Settings;
    let countries: CountryInfo[] | null;
    let loadError: string;
    try {
        [settings, countries, loadError, version] = await Promise.all([
            SettingsService.Get(),
            CountryService.List(),
            CountryService.LoadError(),
            SettingsService.Version(),
        ]);
    } catch (err) {
        void showError(`HowBig could not start: ${err}`);
        return;
    }

    const list = countries ?? [];
    const names = list.map((c) => c.Name);
    const areas = new Map(list.map((c) => [c.Name, c.Area]));
    await document.fonts.ready;
    applySettings(settings, names);

    const leftColor = parseHexColor(settings.left_color);
    const rightColor = parseHexColor(settings.right_color);
    const colors: Record<string, SideColors> = {
        left: {fill: cssColor(leftColor, fillAlpha), border: cssColor(parseHexColor(settings.left_border_color))},
        right: {fill: cssColor(rightColor, fillAlpha), border: cssColor(parseHexColor(settings.right_border_color))},
    };

    const header = byId<HTMLElement>("header");
    const leftBar = byId<HTMLElement>("left-bar");
    const rightBar = byId<HTMLElement>("right-bar");
    const mapBox = byId<HTMLElement>("map");
    const svg = mapBox.querySelector("svg") as SVGSVGElement;

    let left = "";
    let right = "";

    // Errors already shown for the current selection, so a resize doesn't repeat them.
    const shownErrors = new Set<string>();
    const reportError = (msg: string) => {
        if (!shownErrors.has(msg)) {
            shownErrors.add(msg);
            void showError(msg);
        }
    };

    let lastKey = "";
    const layouts = new LatestCall<MapLayout>();

    /** Lays out and draws the map for the current selection and map box size. */
    const refreshMap = () => {
        const w = mapBox.clientWidth;
        const h = mapBox.clientHeight;
        const key = `${left}\n${right}\n${w}x${h}`;
        if (key === lastKey) {
            return;
        }
        lastKey = key;
        layouts.cancel();
        if ((!left && !right) || w <= 0 || h <= 0) {
            renderMap(svg, null, w, h, colors);
            return;
        }
        layouts.run(MapService.Layout(left, right, w, h), (layout) => {
            renderMap(svg, layout, w, h, colors);
            for (const c of layout.countries ?? []) {
                if (c.error) {
                    reportError(c.error);
                }
            }
        }, (err) => {
            lastKey = "";
            reportError(`Error laying out the map: ${err}`);
        });
    };

    const onSelectionChange = () => {
        shownErrors.clear();
        const leftArea = areas.get(left) ?? 0;
        const rightArea = areas.get(right) ?? 0;
        const maxArea = Math.max(leftArea, rightArea);

        const lines: HeaderLine[] = [];
        if (left) {
            lines.push({name: left, area: leftArea, color: cssColor(leftColor)});
        }
        if (right) {
            lines.push({name: right, area: rightArea, color: cssColor(rightColor)});
        }
        renderHeader(header, lines);
        renderBar(leftBar, left !== "", leftArea, maxArea, colors.left.fill);
        renderBar(rightBar, right !== "", rightArea, maxArea, colors.right.fill);
        // The header may have changed height; clientHeight in refreshMap sees the new layout.
        refreshMap();
    };

    createCountryList(byId("left-list"), names, (name) => {
        left = name;
        onSelectionChange();
    });
    createCountryList(byId("right-list"), names, (name) => {
        right = name;
        onSelectionChange();
    });

    let resizeTimer: number | undefined;
    new ResizeObserver(() => {
        window.clearTimeout(resizeTimer);
        resizeTimer = window.setTimeout(refreshMap, resizeDebounceMs);
    }).observe(mapBox);

    // v1.0.0 shows About on launch; a data load error goes on top of it.
    about();
    if (loadError) {
        void showError(loadError);
    }
}

void start();
