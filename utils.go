// SPDX-FileCopyrightText: 2026 Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

// singlePolyLine processes polygon rings and keeps only the outer ring, discarding holes or interior polygons.
// It returns a slice containing only the first ring of the provided polygon structure.
func singlePolyLine(rings [][][]float64) [][][]float64 {
	if len(rings) > 0 {
		return rings[0:1]
	}
	return rings
}
