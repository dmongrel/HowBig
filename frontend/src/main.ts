// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Scaffold only: proves the Go bindings are wired. The real UI comes in a later task.
import {CountryService} from "../bindings/HowBig";

CountryService.List()
    .then((countries) => console.log(`HowBig: ${(countries ?? []).length} countries loaded`))
    .catch((err) => console.error("HowBig: CountryService.List failed", err));
