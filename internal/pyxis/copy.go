package pyxis

import "slices"

// DeepCopy returns an independent copy of in for submission flows.
// It copies nested pointers and slices so changes to the copy do not affect
// the caller's input. It returns nil when in is nil.
func (in *CertificationInput) DeepCopy() *CertificationInput {
	if in == nil {
		return nil
	}

	// Copy value fields first, then replace shared pointers and slices below.
	out := *in
	out.CertProject = cloneCertProject(in.CertProject)
	out.CertImage = cloneCertImage(in.CertImage)
	out.TestResults = cloneTestResults(in.TestResults)
	out.RpmManifest = cloneRPMManifest(in.RpmManifest)
	// Artifacts are value structs, but the outer slice still needs new storage.
	out.Artifacts = slices.Clone(in.Artifacts)

	return &out
}

func cloneCertProject(in *CertProject) *CertProject {
	if in == nil {
		return nil
	}

	out := *in
	return &out
}

func cloneCertImage(in *CertImage) *CertImage {
	if in == nil {
		return nil
	}

	out := *in
	// Give the copied image its own repository backing array.
	out.Repositories = slices.Clone(in.Repositories)
	for i := range out.Repositories {
		// Each repository owns a nested tag slice.
		out.Repositories[i].Tags = slices.Clone(in.Repositories[i].Tags)
	}

	// Freshness grades are values, but the slice backing array is shared by a
	// shallow copy.
	out.FreshnessGrades = slices.Clone(in.FreshnessGrades)

	if in.ParsedData != nil {
		// ParsedData is a pointer with several slices of its own.
		parsedData := *in.ParsedData
		parsedData.Labels = slices.Clone(in.ParsedData.Labels)
		parsedData.Layers = slices.Clone(in.ParsedData.Layers)
		parsedData.UncompressedLayerSizes = slices.Clone(in.ParsedData.UncompressedLayerSizes)
		out.ParsedData = &parsedData
	}

	return &out
}

func cloneTestResults(in *TestResults) *TestResults {
	if in == nil {
		return nil
	}

	out := *in
	// UserResponse contains slices that must not share backing arrays.
	out.Results.Passed = slices.Clone(in.Results.Passed)
	out.Results.Failed = slices.Clone(in.Results.Failed)
	out.Results.Errors = slices.Clone(in.Results.Errors)
	out.Results.Warnings = slices.Clone(in.Results.Warnings)

	return &out
}

func cloneRPMManifest(in *RPMManifest) *RPMManifest {
	if in == nil {
		return nil
	}

	out := *in
	// RPMs are value elements, but their slice backing array is still shared by
	// a shallow copy.
	out.RPMS = slices.Clone(in.RPMS)

	return &out
}
