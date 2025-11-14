package router

import (
	"reflect"
	"testing"
)

func TestConcatAxis0_Int32(t *testing.T) {
	x := []interface{}{
		[][]int32{{1}, {2}},
	}
	y := []interface{}{
		[][]int32{{3}},
	}

	got, err := concatAxis0(x, y)
	if err != nil {
		t.Fatalf("concatAxis0 returned error: %v", err)
	}

	want := []interface{}{
		[][]int32{{1}, {2}, {3}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("concatAxis0() mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestConcatAxis0_String(t *testing.T) {
	x := []interface{}{
		[][]string{{"a"}},
	}
	y := []interface{}{
		[][]string{{"b"}, {"c"}},
	}

	got, err := concatAxis0(x, y)
	if err != nil {
		t.Fatalf("concatAxis0 returned error: %v", err)
	}

	want := []interface{}{
		[][]string{{"a"}, {"b"}, {"c"}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("concatAxis0() mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestConcatAxis0_TypeMismatch(t *testing.T) {
	x := []interface{}{
		[][]int64{{1}},
	}
	y := []interface{}{
		[][]int32{{2}},
	}

	_, err := concatAxis0(x, y)
	if err == nil {
		t.Fatalf("expected type mismatch error, got nil")
	}
}

func TestDebatchAndSqueezeBatch_Int64(t *testing.T) {
	// batch of 3, single-column
	batch := [][]int64{{10}, {20}, {30}}

	// pick index 1
	debatched, err := debatch(batch, 1)
	if err != nil {
		t.Fatalf("debatch returned error: %v", err)
	}

	wantDebatched := [][]int64{{20}}
	if !reflect.DeepEqual(debatched, wantDebatched) {
		t.Errorf("debatch() mismatch:\n got: %#v\nwant: %#v", debatched, wantDebatched)
	}

	// squeeze should return the scalar 20
	scalar, err := squeezeBatch(debatched)
	if err != nil {
		t.Fatalf("squeezeBatch returned error: %v", err)
	}

	if v, ok := scalar.(int64); !ok || v != 20 {
		t.Errorf("squeezeBatch() got %v (%T), want 20 (int64)", scalar, scalar)
	}
}
