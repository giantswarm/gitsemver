package main

import (
	"errors"
	"testing"
)

func Test_runValidate_valid(t *testing.T) {
	t.Parallel()
	if err := runValidate([]string{"1.2.3"}); err != nil {
		t.Errorf("runValidate(1.2.3) = %v, want nil", err)
	}
}

func Test_runValidate_invalid(t *testing.T) {
	t.Parallel()
	if err := runValidate([]string{"not-a-version"}); !errors.Is(err, errInvalidVersion) {
		t.Errorf("runValidate(not-a-version) = %v, want errInvalidVersion", err)
	}
}

func Test_runValidate_typeFlag(t *testing.T) {
	t.Parallel()
	cases := []struct {
		args    []string
		wantErr bool
	}{
		{[]string{"--type", "stable", "1.2.3"}, false},
		{[]string{"--type", "rc", "1.2.3-rc.1"}, false},
		{[]string{"--type", "stable", "1.2.3-rc.1"}, true},
		{[]string{"--type", "unknown", "1.2.3"}, true},
	}
	for _, tc := range cases {
		got := runValidate(tc.args)
		if (got != nil) != tc.wantErr {
			t.Errorf("runValidate(%v) err=%v, wantErr=%v", tc.args, got, tc.wantErr)
		}
	}
}

func Test_runNext_lastTag_patch(t *testing.T) {
	t.Parallel()
	if err := runNext([]string{"patch", "--last-tag", "v1.2.3"}); err != nil {
		t.Errorf("runNext patch --last-tag v1.2.3 = %v, want nil", err)
	}
}

func Test_runNext_lastTag_rc(t *testing.T) {
	t.Parallel()
	if err := runNext([]string{"rc", "--last-tag", "v1.2.3-rc.1"}); err != nil {
		t.Errorf("runNext rc --last-tag v1.2.3-rc.1 = %v, want nil", err)
	}
}

func Test_runNext_missingBumpType(t *testing.T) {
	t.Parallel()
	if err := runNext([]string{}); err == nil {
		t.Error("runNext with no args should return error")
	}
}

func Test_runNext_unknownBumpType(t *testing.T) {
	t.Parallel()
	if err := runNext([]string{"bogus", "--last-tag", "v1.2.3"}); err == nil {
		t.Error("runNext with unknown bump type should return error")
	}
}
