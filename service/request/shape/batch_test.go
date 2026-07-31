package shape

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

	got, err := ConcatAxis0(x, y)
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

	got, err := ConcatAxis0(x, y)
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

	_, err := ConcatAxis0(x, y)
	if err == nil {
		t.Fatalf("expected type mismatch error, got nil")
	}
}

func TestDebatchAndSqueezeBatch_Int64(t *testing.T) {
	// batch of 3, single-column
	batch := [][]int64{{10}, {20}, {30}}

	// pick index 1
	debatched, err := Debatch(batch, 1)
	if err != nil {
		t.Fatalf("debatch returned error: %v", err)
	}

	wantDebatched := [][]int64{{20}}
	if !reflect.DeepEqual(debatched, wantDebatched) {
		t.Errorf("debatch() mismatch:\n got: %#v\nwant: %#v", debatched, wantDebatched)
	}

	// squeeze should return the scalar 20
	scalar, err := SqueezeBatch(debatched)
	if err != nil {
		t.Fatalf("squeezeBatch returned error: %v", err)
	}

	if v, ok := scalar.(int64); !ok || v != 20 {
		t.Errorf("squeezeBatch() got %v (%T), want 20 (int64)", scalar, scalar)
	}
}

func TestAppendRowToBatch_String(t *testing.T) {
	// Start with nil accumulator
	row1 := [][]string{{"hello"}}
	acc, err := AppendRowToBatch(nil, row1)
	if err != nil {
		t.Fatalf("AppendRowToBatch returned error: %v", err)
	}

	want1 := [][]string{{"hello"}}
	if !reflect.DeepEqual(acc, want1) {
		t.Errorf("after first append: got %#v, want %#v", acc, want1)
	}

	// Append second row
	row2 := [][]string{{"world"}}
	acc, err = AppendRowToBatch(acc, row2)
	if err != nil {
		t.Fatalf("AppendRowToBatch returned error: %v", err)
	}

	want2 := [][]string{{"hello"}, {"world"}}
	if !reflect.DeepEqual(acc, want2) {
		t.Errorf("after second append: got %#v, want %#v", acc, want2)
	}

	// Append third row
	row3 := [][]string{{"foo"}}
	acc, err = AppendRowToBatch(acc, row3)
	if err != nil {
		t.Fatalf("AppendRowToBatch returned error: %v", err)
	}

	want3 := [][]string{{"hello"}, {"world"}, {"foo"}}
	if !reflect.DeepEqual(acc, want3) {
		t.Errorf("after third append: got %#v, want %#v", acc, want3)
	}
}

func TestAppendRowToBatch_Float32(t *testing.T) {
	row1 := [][]float32{{1.5}}
	acc, err := AppendRowToBatch(nil, row1)
	if err != nil {
		t.Fatalf("AppendRowToBatch returned error: %v", err)
	}

	row2 := [][]float32{{2.5}}
	acc, err = AppendRowToBatch(acc, row2)
	if err != nil {
		t.Fatalf("AppendRowToBatch returned error: %v", err)
	}

	want := [][]float32{{1.5}, {2.5}}
	if !reflect.DeepEqual(acc, want) {
		t.Errorf("got %#v, want %#v", acc, want)
	}
}

func TestAppendRowToBatch_TypeMismatch(t *testing.T) {
	acc := [][]string{{"hello"}}
	row := [][]int64{{123}}

	_, err := AppendRowToBatch(acc, row)
	if err == nil {
		t.Fatal("expected type mismatch error, got nil")
	}
}

func TestExtractRowFromBatch_String(t *testing.T) {
	batch := [][]string{{"a"}, {"b"}, {"c"}}

	row, err := ExtractRowFromBatch(batch, 1)
	if err != nil {
		t.Fatalf("ExtractRowFromBatch returned error: %v", err)
	}

	want := [][]string{{"b"}}
	if !reflect.DeepEqual(row, want) {
		t.Errorf("got %#v, want %#v", row, want)
	}
}

func TestExtractRowFromBatch_Float32(t *testing.T) {
	batch := [][]float32{{1.0, 2.0}, {3.0, 4.0}, {5.0, 6.0}}

	row, err := ExtractRowFromBatch(batch, 2)
	if err != nil {
		t.Fatalf("ExtractRowFromBatch returned error: %v", err)
	}

	want := [][]float32{{5.0, 6.0}}
	if !reflect.DeepEqual(row, want) {
		t.Errorf("got %#v, want %#v", row, want)
	}
}

func TestExtractRowFromBatch_OutOfRange(t *testing.T) {
	batch := [][]int64{{1}, {2}}

	_, err := ExtractRowFromBatch(batch, 5)
	if err == nil {
		t.Fatal("expected out of range error, got nil")
	}
}

func TestBatchSize(t *testing.T) {
	tests := []struct {
		name  string
		batch interface{}
		want  int
	}{
		{"string batch", [][]string{{"a"}, {"b"}, {"c"}}, 3},
		{"float32 batch", [][]float32{{1}, {2}}, 2},
		{"int64 batch", [][]int64{{1}}, 1},
		{"empty batch", [][]string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BatchSize(tt.batch)
			if err != nil {
				t.Fatalf("BatchSize returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
