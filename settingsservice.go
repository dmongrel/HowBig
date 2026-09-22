// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import "HowBig/internal/settings"

// SettingsService exposes the loaded settings.json to the frontend.
type SettingsService struct {
	settings settings.Settings
}

// NewSettingsService copies s so later changes to it don't leak to the frontend.
func NewSettingsService(s *settings.Settings) *SettingsService {
	return &SettingsService{settings: *s}
}

// Get returns the settings as loaded, with defaults filled in.
func (s *SettingsService) Get() settings.Settings {
	return s.settings
}

// Version returns the app version shown in the About dialog.
func (s *SettingsService) Version() string {
	return Version
}
