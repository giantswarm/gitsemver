package gitrepo

import "testing"

func TestIsValidStable(t *testing.T) {
	t.Parallel()

	valid := []string{
		"1.2.3",
		"0.0.0",
		"10.20.30",
		"v1.2.3",
		"v0.0.0",
		"v10.20.30",
	}
	invalid := []string{
		"",
		"1.2",
		"1.2.3.4",
		"1.2.3-rc.1",
		"v1.2.3-rc.1",
		"1.2.3-dev.main.2026-01-27.09-49-59",
		"v1.2.3-dev.main.2026-01-27.09-49-59",
		"1.2.3-anything",
		"abc",
		"v",
		"1.2.x",
	}

	for _, v := range valid {
		if !IsValidStable(v) {
			t.Errorf("IsValidStable(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if IsValidStable(v) {
			t.Errorf("IsValidStable(%q) = true, want false", v)
		}
	}
}

func TestIsValidRC(t *testing.T) {
	t.Parallel()

	valid := []string{
		"1.2.3-rc.1",
		"1.2.3-rc.0",
		"1.2.3-rc.42",
		"v1.2.3-rc.1",
		"v0.0.1-rc.99",
	}
	invalid := []string{
		"",
		"1.2.3",
		"v1.2.3",
		"1.2.3-rc",
		"1.2.3-rc.",
		"1.2.3-rc.a",
		"1.2.3-RC.1",
		"1.2.3-rc.1.extra",
		"1.2.3-dev.main.2026-01-27.09-49-59",
		"abc",
	}

	for _, v := range valid {
		if !IsValidRC(v) {
			t.Errorf("IsValidRC(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if IsValidRC(v) {
			t.Errorf("IsValidRC(%q) = true, want false", v)
		}
	}
}

func TestIsValidDev(t *testing.T) {
	t.Parallel()

	valid := []string{
		"1.2.3-dev.main.2026-01-27.09-49-59",
		"0.0.0-dev.main.2026-01-27.09-49-59",
		"1.2.4-dev.my-feature.2026-01-27.09-49-59",
		"v1.2.3-dev.main.2026-01-27.09-49-59",
		"v0.0.0-dev.main.2026-01-27.09-49-59",
		"2.0.1-dev.test-branch.2026-01-27.09-49-59",
		// branch names with hyphens
		"1.0.1-dev.feature-my-thing.2026-05-21.14-30-00",
		// single-char branch
		"1.0.0-dev.x.2026-05-21.00-00-00",
	}
	invalid := []string{
		"",
		"1.2.3",
		"1.2.3-rc.1",
		// missing time segment
		"1.2.3-dev.main.2026-01-27",
		// wrong date separator (slashes)
		"1.2.3-dev.main.2026/01/27.09-49-59",
		// wrong time separator (colons)
		"1.2.3-dev.main.2026-01-27.09:49:59",
		// date wrong length
		"1.2.3-dev.main.26-01-27.09-49-59",
		// branch with slash (not sanitized)
		"1.2.3-dev.feat/foo.2026-01-27.09-49-59",
		// empty branch
		"1.2.3-dev..2026-01-27.09-49-59",
		// extra trailing segment
		"1.2.3-dev.main.2026-01-27.09-49-59.extra",
		// no v prefix required but "vv" is wrong
		"vv1.2.3-dev.main.2026-01-27.09-49-59",
	}

	for _, v := range valid {
		if !IsValidDev(v) {
			t.Errorf("IsValidDev(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if IsValidDev(v) {
			t.Errorf("IsValidDev(%q) = true, want false", v)
		}
	}
}

func TestIsValid(t *testing.T) {
	t.Parallel()

	valid := []string{
		// stable
		"1.2.3",
		"v1.2.3",
		// RC
		"1.2.3-rc.1",
		"v1.2.3-rc.1",
		// dev
		"1.2.4-dev.main.2026-01-27.09-49-59",
		"v1.2.4-dev.main.2026-01-27.09-49-59",
	}
	invalid := []string{
		"",
		"abc",
		"1.2",
		"1.2.3.4",
		"1.2.3-",
		"1.2.3-rc",
		"1.2.3-dev.main.2026-01-27",
	}

	for _, v := range valid {
		if !IsValid(v) {
			t.Errorf("IsValid(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if IsValid(v) {
			t.Errorf("IsValid(%q) = true, want false", v)
		}
	}
}
