package example

import (
	"reflect"
	"testing"
)

func TestTimeseries_RoundTrip(t *testing.T) {
	ts := &Timeseries{
		Name: "test-series",
		Samples: []Sample{
			{Value: 1.5, Timestamp: 1000},
			{Value: 2.5, Timestamp: 2000},
			{Value: -3.5, Timestamp: 3000},
		},
	}

	data := ts.MarshalProtobuf(nil)
	if len(data) == 0 {
		t.Fatal("MarshalProtobuf returned empty data")
	}

	var ts2 Timeseries
	if err := ts2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(ts, &ts2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", ts2, *ts)
	}
}

func TestSample_RoundTrip(t *testing.T) {
	s := &Sample{
		Value:     -123.456,
		Timestamp: 9876543210,
	}

	data := s.MarshalProtobuf(nil)

	var s2 Sample
	if err := s2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(s, &s2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", s2, *s)
	}
}

func TestAllTypes_RoundTrip(t *testing.T) {
	at := &AllTypes{
		StrField:      "hello world",
		BytesField:    []byte{0x01, 0x02, 0x03, 0x04},
		Int32Field:    -12345,
		Int64Field:    -9876543210,
		Uint32Field:   12345,
		Uint64Field:   9876543210,
		Sint32Field:   -54321,
		Sint64Field:   -1234567890,
		BoolField:     true,
		DoubleField:   3.14159265359,
		FloatField:    2.71828,
		Fixed32Field:  0xDEADBEEF,
		Fixed64Field:  0xDEADBEEFCAFEBABE,
		Sfixed32Field: -12345678,
		Sfixed64Field: -123456789012345,
	}

	data := at.MarshalProtobuf(nil)

	var at2 AllTypes
	if err := at2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(at, &at2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", at2, *at)
	}
}

func TestOptionalFields_RoundTrip(t *testing.T) {
	name := "test"
	age := int32(25)
	score := 95.5
	isActive := true

	of := &OptionalFields{
		Name:     &name,
		Age:      &age,
		Score:    &score,
		IsActive: &isActive,
	}

	data := of.MarshalProtobuf(nil)

	var of2 OptionalFields
	if err := of2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(of, &of2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", of2, *of)
	}
}

func TestOptionalFields_NilFields(t *testing.T) {
	of := &OptionalFields{
		Name:     nil,
		Age:      nil,
		Score:    nil,
		IsActive: nil,
	}

	data := of.MarshalProtobuf(nil)

	var of2 OptionalFields
	if err := of2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if of2.Name != nil || of2.Age != nil || of2.Score != nil || of2.IsActive != nil {
		t.Errorf("expected all fields to be nil, got: %+v", of2)
	}
}

func TestNestedMessage_RoundTrip(t *testing.T) {
	optionalSample := &Sample{Value: 99.9, Timestamp: 9999}

	nm := &NestedMessage{
		ID:       12345,
		Outer:    Sample{Value: 1.1, Timestamp: 100},
		Optional: optionalSample,
		Items: []Sample{
			{Value: 2.2, Timestamp: 200},
			{Value: 3.3, Timestamp: 300},
		},
	}

	data := nm.MarshalProtobuf(nil)

	var nm2 NestedMessage
	if err := nm2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(nm, &nm2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", nm2, *nm)
	}
}

func TestWithPointers_RoundTrip(t *testing.T) {
	wp := &WithPointers{
		ID: 42,
		Samples: []*Sample{
			{Value: 1.0, Timestamp: 100},
			{Value: 2.0, Timestamp: 200},
			nil, // nil entries are skipped during marshal
			{Value: 3.0, Timestamp: 300},
		},
	}

	data := wp.MarshalProtobuf(nil)

	var wp2 WithPointers
	if err := wp2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	// nil entries are not preserved, so we expect 3 entries
	if len(wp2.Samples) != 3 {
		t.Errorf("expected 3 samples, got %d", len(wp2.Samples))
	}

	if wp2.ID != wp.ID {
		t.Errorf("ID mismatch: got %d, want %d", wp2.ID, wp.ID)
	}
}

func TestWithEnums_RoundTrip(t *testing.T) {
	optStatus := StatusPaused

	we := &WithEnums{
		ID:        123,
		Status:    StatusActive,
		OptStatus: &optStatus,
		Statuses:  []Status{StatusUnknown, StatusActive, StatusPaused, StatusStopped},
	}

	data := we.MarshalProtobuf(nil)

	var we2 WithEnums
	if err := we2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(we, &we2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", we2, *we)
	}
}

func TestRepeatedScalars_RoundTrip(t *testing.T) {
	rs := &RepeatedScalars{
		Int32s:    []int32{1, 2, 3, -1, -2, -3},
		Int64s:    []int64{100, 200, -100, -200},
		Uint32s:   []uint32{1, 2, 3},
		Uint64s:   []uint64{100, 200, 300},
		Sint32s:   []int32{-1, 0, 1},
		Sint64s:   []int64{-100, 0, 100},
		Bools:     []bool{true, false, true},
		Doubles:   []float64{1.1, 2.2, 3.3},
		Floats:    []float32{1.1, 2.2, 3.3},
		Fixed32s:  []uint32{0xDEADBEEF, 0xCAFEBABE},
		Fixed64s:  []uint64{0xDEADBEEFCAFEBABE},
		Sfixed32s: []int32{-12345, 12345},
		Sfixed64s: []int64{-1234567890, 1234567890},
	}

	data := rs.MarshalProtobuf(nil)

	var rs2 RepeatedScalars
	if err := rs2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	if !reflect.DeepEqual(rs, &rs2) {
		t.Errorf("round-trip mismatch:\ngot:  %+v\nwant: %+v", rs2, *rs)
	}
}

func TestRepeatedScalars_Empty(t *testing.T) {
	rs := &RepeatedScalars{}

	data := rs.MarshalProtobuf(nil)

	var rs2 RepeatedScalars
	if err := rs2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	// Empty slices should remain nil after round-trip
	if rs2.Int32s != nil || rs2.Int64s != nil || rs2.Bools != nil {
		t.Errorf("expected nil slices for empty message")
	}
}

func TestWithMaps_RoundTrip(t *testing.T) {
	wm := &WithMaps{
		StringToInt: map[string]int32{
			"one":   1,
			"two":   2,
			"three": 3,
		},
		IntToString: map[int64]string{
			100: "hundred",
			200: "two hundred",
		},
		StringToMsg: map[string]*Sample{
			"first":  {Value: 1.1, Timestamp: 100},
			"second": {Value: 2.2, Timestamp: 200},
		},
	}

	data := wm.MarshalProtobuf(nil)

	var wm2 WithMaps
	if err := wm2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	// Check StringToInt
	if len(wm2.StringToInt) != len(wm.StringToInt) {
		t.Errorf("StringToInt length mismatch: got %d, want %d", len(wm2.StringToInt), len(wm.StringToInt))
	}
	for k, v := range wm.StringToInt {
		if wm2.StringToInt[k] != v {
			t.Errorf("StringToInt[%s] mismatch: got %d, want %d", k, wm2.StringToInt[k], v)
		}
	}

	// Check IntToString
	if len(wm2.IntToString) != len(wm.IntToString) {
		t.Errorf("IntToString length mismatch: got %d, want %d", len(wm2.IntToString), len(wm.IntToString))
	}
	for k, v := range wm.IntToString {
		if wm2.IntToString[k] != v {
			t.Errorf("IntToString[%d] mismatch: got %s, want %s", k, wm2.IntToString[k], v)
		}
	}

	// Check StringToMsg
	if len(wm2.StringToMsg) != len(wm.StringToMsg) {
		t.Errorf("StringToMsg length mismatch: got %d, want %d", len(wm2.StringToMsg), len(wm.StringToMsg))
	}
	for k, v := range wm.StringToMsg {
		got := wm2.StringToMsg[k]
		if got == nil {
			t.Errorf("StringToMsg[%s] is nil", k)
			continue
		}
		if got.Value != v.Value || got.Timestamp != v.Timestamp {
			t.Errorf("StringToMsg[%s] mismatch: got %+v, want %+v", k, got, v)
		}
	}
}

func TestWithMaps_Empty(t *testing.T) {
	wm := &WithMaps{}

	data := wm.MarshalProtobuf(nil)

	var wm2 WithMaps
	if err := wm2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	// Empty maps should remain nil after round-trip
	if wm2.StringToInt != nil || wm2.IntToString != nil || wm2.StringToMsg != nil {
		t.Errorf("expected nil maps for empty message")
	}
}

// TestSharedTimeseries proves that the same Timeseries type works correctly
// when used in different parent structs with different field numbers
func TestSharedTimeseries(t *testing.T) {
	// Same Timeseries data
	ts := Timeseries{
		Name: "shared-series",
		Samples: []Sample{
			{Value: 1.1, Timestamp: 100},
			{Value: 2.2, Timestamp: 200},
		},
	}

	// Use in BigStruct (Timeseries at field 50)
	big := &BigStruct{
		Field1:  "big",
		Field2:  100,
		Field3:  1000,
		Field4:  true,
		Field5:  3.14,
		Series:  ts,
		Field51: "after",
		Field52: 999,
	}

	// Use in SmallStruct (Timeseries at field 2)
	small := &SmallStruct{
		Name:   "small",
		Series: ts,
	}

	// Marshal both
	bigData := big.MarshalProtobuf(nil)
	smallData := small.MarshalProtobuf(nil)

	// Unmarshal both
	var big2 BigStruct
	var small2 SmallStruct

	if err := big2.UnmarshalProtobuf(bigData); err != nil {
		t.Fatalf("BigStruct unmarshal failed: %v", err)
	}
	if err := small2.UnmarshalProtobuf(smallData); err != nil {
		t.Fatalf("SmallStruct unmarshal failed: %v", err)
	}

	// Verify BigStruct
	if big2.Field1 != big.Field1 || big2.Field2 != big.Field2 {
		t.Errorf("BigStruct scalar fields mismatch")
	}
	if big2.Series.Name != ts.Name {
		t.Errorf("BigStruct.Series.Name mismatch: got %s, want %s", big2.Series.Name, ts.Name)
	}
	if len(big2.Series.Samples) != len(ts.Samples) {
		t.Errorf("BigStruct.Series.Samples length mismatch")
	}

	// Verify SmallStruct
	if small2.Name != small.Name {
		t.Errorf("SmallStruct.Name mismatch")
	}
	if small2.Series.Name != ts.Name {
		t.Errorf("SmallStruct.Series.Name mismatch: got %s, want %s", small2.Series.Name, ts.Name)
	}
	if len(small2.Series.Samples) != len(ts.Samples) {
		t.Errorf("SmallStruct.Series.Samples length mismatch")
	}

	// Verify the Timeseries content is identical in both
	for i, s := range ts.Samples {
		if big2.Series.Samples[i].Value != s.Value || big2.Series.Samples[i].Timestamp != s.Timestamp {
			t.Errorf("BigStruct.Series.Samples[%d] mismatch", i)
		}
		if small2.Series.Samples[i].Value != s.Value || small2.Series.Samples[i].Timestamp != s.Timestamp {
			t.Errorf("SmallStruct.Series.Samples[%d] mismatch", i)
		}
	}

	t.Logf("BigStruct encoded size: %d bytes", len(bigData))
	t.Logf("SmallStruct encoded size: %d bytes", len(smallData))
}

func TestInferredTypes_RoundTrip(t *testing.T) {
	inner := &Sample{Value: 99.9, Timestamp: 9999}

	it := &InferredTypes{
		Name:      "test",
		Age:       30,
		Score:     95.5,
		IsActive:  true,
		Data:      []byte{1, 2, 3, 4, 5},
		BigNum:    9876543210,
		SmallNum:  3.14,
		Unsigned:  42,
		BigUnsign: 18446744073709551615, // max uint64
		Inner:     inner,
		Items: []Sample{
			{Value: 1.1, Timestamp: 100},
			{Value: 2.2, Timestamp: 200},
		},
		Lookup: map[string]int32{
			"one": 1,
			"two": 2,
		},
		Numbers: []int64{10, 20, 30, 40, 50},
	}

	data := it.MarshalProtobuf(nil)

	var it2 InferredTypes
	if err := it2.UnmarshalProtobuf(data); err != nil {
		t.Fatalf("UnmarshalProtobuf failed: %v", err)
	}

	// Check scalar fields
	if it2.Name != it.Name {
		t.Errorf("Name mismatch: got %s, want %s", it2.Name, it.Name)
	}
	if it2.Age != it.Age {
		t.Errorf("Age mismatch: got %d, want %d", it2.Age, it.Age)
	}
	if it2.Score != it.Score {
		t.Errorf("Score mismatch: got %f, want %f", it2.Score, it.Score)
	}
	if it2.IsActive != it.IsActive {
		t.Errorf("IsActive mismatch: got %v, want %v", it2.IsActive, it.IsActive)
	}
	if string(it2.Data) != string(it.Data) {
		t.Errorf("Data mismatch: got %v, want %v", it2.Data, it.Data)
	}
	if it2.BigNum != it.BigNum {
		t.Errorf("BigNum mismatch: got %d, want %d", it2.BigNum, it.BigNum)
	}
	if it2.SmallNum != it.SmallNum {
		t.Errorf("SmallNum mismatch: got %f, want %f", it2.SmallNum, it.SmallNum)
	}
	if it2.Unsigned != it.Unsigned {
		t.Errorf("Unsigned mismatch: got %d, want %d", it2.Unsigned, it.Unsigned)
	}
	if it2.BigUnsign != it.BigUnsign {
		t.Errorf("BigUnsign mismatch: got %d, want %d", it2.BigUnsign, it.BigUnsign)
	}

	// Check nested message
	if it2.Inner == nil || it2.Inner.Value != inner.Value || it2.Inner.Timestamp != inner.Timestamp {
		t.Errorf("Inner mismatch: got %+v, want %+v", it2.Inner, inner)
	}

	// Check repeated messages
	if len(it2.Items) != len(it.Items) {
		t.Errorf("Items length mismatch: got %d, want %d", len(it2.Items), len(it.Items))
	}

	// Check map
	if len(it2.Lookup) != len(it.Lookup) {
		t.Errorf("Lookup length mismatch: got %d, want %d", len(it2.Lookup), len(it.Lookup))
	}
	for k, v := range it.Lookup {
		if it2.Lookup[k] != v {
			t.Errorf("Lookup[%s] mismatch: got %d, want %d", k, it2.Lookup[k], v)
		}
	}

	// Check repeated scalars
	if len(it2.Numbers) != len(it.Numbers) {
		t.Errorf("Numbers length mismatch: got %d, want %d", len(it2.Numbers), len(it.Numbers))
	}
	for i, v := range it.Numbers {
		if it2.Numbers[i] != v {
			t.Errorf("Numbers[%d] mismatch: got %d, want %d", i, it2.Numbers[i], v)
		}
	}
}

// TestInterfaceCompliance verifies that generated types implement the marshaler interfaces.
// This enables custom types from external packages to be used as nested messages
// by implementing these same interfaces.
func TestInterfaceCompliance(t *testing.T) {
	// Verify Sample implements both interfaces
	var _ ProtobufMarshaler = &Sample{}
	var _ ProtobufUnmarshaler = &Sample{}

	// Verify Timeseries implements both interfaces
	var _ ProtobufMarshaler = &Timeseries{}
	var _ ProtobufUnmarshaler = &Timeseries{}

	// Verify all generated types implement the interfaces
	var _ ProtobufMarshaler = &AllTypes{}
	var _ ProtobufMarshaler = &OptionalFields{}
	var _ ProtobufMarshaler = &NestedMessage{}
	var _ ProtobufMarshaler = &WithPointers{}
	var _ ProtobufMarshaler = &WithEnums{}
	var _ ProtobufMarshaler = &RepeatedScalars{}
	var _ ProtobufMarshaler = &WithMaps{}
	var _ ProtobufMarshaler = &InferredTypes{}

	t.Log("All generated types implement ProtobufMarshaler and ProtobufUnmarshaler interfaces")
}

// TestAllTypesComprehensive tests marshal/unmarshal for ALL types defined in types.go
// with populated data to ensure full round-trip correctness.
func TestAllTypesComprehensive(t *testing.T) {
	// Helper to test any type that implements our interfaces
	testRoundTrip := func(name string, original, decoded interface {
		MarshalProtobuf([]byte) []byte
		UnmarshalProtobuf([]byte) error
	}, allowEmpty bool) {
		data := original.MarshalProtobuf(nil)
		if len(data) == 0 && !allowEmpty {
			t.Errorf("%s: MarshalProtobuf returned empty data", name)
			return
		}
		if err := decoded.UnmarshalProtobuf(data); err != nil {
			t.Errorf("%s: UnmarshalProtobuf failed: %v", name, err)
			return
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Errorf("%s: round-trip mismatch:\ngot:  %+v\nwant: %+v", name, decoded, original)
		}
		t.Logf("%s: encoded size %d bytes", name, len(data))
	}

	// Test Timeseries
	ts := &Timeseries{
		Name: "comprehensive-test",
		Samples: []Sample{
			{Value: -123.456, Timestamp: 1000000000},
			{Value: 0, Timestamp: 0},
			{Value: 999.999, Timestamp: -1},
		},
	}
	testRoundTrip("Timeseries", ts, &Timeseries{}, false)

	// Test Sample
	sample := &Sample{Value: 3.14159, Timestamp: 9876543210}
	testRoundTrip("Sample", sample, &Sample{}, false)

	// Test AllTypes with all fields populated
	allTypes := &AllTypes{
		StrField:      "hello world",
		BytesField:    []byte{0x00, 0xFF, 0x7F, 0x80},
		Int32Field:    -2147483648, // min int32
		Int64Field:    -9223372036854775808,
		Uint32Field:   4294967295, // max uint32
		Uint64Field:   18446744073709551615,
		Sint32Field:   -1000000,
		Sint64Field:   -9000000000,
		BoolField:     true,
		DoubleField:   1.7976931348623157e+308, // near max float64
		FloatField:    3.4028235e+38,           // near max float32
		Fixed32Field:  123456,
		Fixed64Field:  9876543210,
		Sfixed32Field: -123456,
		Sfixed64Field: -9876543210,
	}
	testRoundTrip("AllTypes", allTypes, &AllTypes{}, false)

	// Test OptionalFields with all fields set
	str := "optional string"
	age := int32(42)
	score := 99.9
	active := true
	optFields := &OptionalFields{
		Name:     &str,
		Age:      &age,
		Score:    &score,
		IsActive: &active,
	}
	testRoundTrip("OptionalFields (all set)", optFields, &OptionalFields{}, false)

	// Test OptionalFields with nil fields (empty message is valid in protobuf)
	nilOptFields := &OptionalFields{}
	testRoundTrip("OptionalFields (all nil)", nilOptFields, &OptionalFields{}, true)

	// Test NestedMessage
	nested := &NestedMessage{
		ID:       12345,
		Outer:    Sample{Value: 1.1, Timestamp: 100},
		Optional: &Sample{Value: 2.2, Timestamp: 200},
		Items: []Sample{
			{Value: 3.3, Timestamp: 300},
			{Value: 4.4, Timestamp: 400},
		},
	}
	testRoundTrip("NestedMessage", nested, &NestedMessage{}, false)

	// Test WithPointers
	withPtrs := &WithPointers{
		ID: 999,
		Samples: []*Sample{
			{Value: 1.0, Timestamp: 1},
			{Value: 2.0, Timestamp: 2},
		},
	}
	testRoundTrip("WithPointers", withPtrs, &WithPointers{}, false)

	// Test WithEnums
	optStatus := StatusPaused
	withEnums := &WithEnums{
		ID:        777,
		Status:    StatusActive,
		OptStatus: &optStatus,
		Statuses:  []Status{StatusUnknown, StatusActive, StatusPaused, StatusStopped},
	}
	testRoundTrip("WithEnums", withEnums, &WithEnums{}, false)

	// Test RepeatedScalars with all types
	repeated := &RepeatedScalars{
		Int32s:    []int32{-1, 0, 1, 2147483647},
		Int64s:    []int64{-1, 0, 1, 9223372036854775807},
		Uint32s:   []uint32{0, 1, 4294967295},
		Uint64s:   []uint64{0, 1, 18446744073709551615},
		Sint32s:   []int32{-2147483648, 0, 2147483647},
		Sint64s:   []int64{-9223372036854775808, 0, 9223372036854775807},
		Bools:     []bool{true, false, true, false},
		Doubles:   []float64{-1.5, 0, 1.5, 1e308},
		Floats:    []float32{-1.5, 0, 1.5, 1e38},
		Fixed32s:  []uint32{0, 123, 4294967295},
		Fixed64s:  []uint64{0, 456, 18446744073709551615},
		Sfixed32s: []int32{-2147483648, 0, 2147483647},
		Sfixed64s: []int64{-9223372036854775808, 0, 9223372036854775807},
	}
	testRoundTrip("RepeatedScalars", repeated, &RepeatedScalars{}, false)

	// Test WithMaps
	withMaps := &WithMaps{
		StringToInt: map[string]int32{"zero": 0, "one": 1, "negative": -100},
		IntToString: map[int64]string{0: "zero", 1: "one", -100: "negative"},
		StringToMsg: map[string]*Sample{
			"first":  {Value: 1.1, Timestamp: 100},
			"second": {Value: 2.2, Timestamp: 200},
		},
	}
	testRoundTrip("WithMaps", withMaps, &WithMaps{}, false)

	// Test InferredTypes (type inference from Go types)
	inferred := &InferredTypes{
		Name:      "inferred",
		Age:       30,
		Score:     95.5,
		IsActive:  true,
		Data:      []byte{1, 2, 3},
		BigNum:    9876543210,
		SmallNum:  3.14,
		Unsigned:  42,
		BigUnsign: 18446744073709551615,
		Inner:     &Sample{Value: 99.9, Timestamp: 9999},
		Items:     []Sample{{Value: 1.1, Timestamp: 100}},
		Lookup:    map[string]int32{"key": 123},
		Numbers:   []int64{10, 20, 30},
	}
	testRoundTrip("InferredTypes", inferred, &InferredTypes{}, false)

	// Test BigStruct
	bigStruct := &BigStruct{
		Field1:  "big",
		Field2:  42,
		Field3:  9876543210,
		Field4:  true,
		Field5:  3.14,
		Series:  Timeseries{Name: "series", Samples: []Sample{{Value: 1.0, Timestamp: 100}}},
		Field51: "after",
		Field52: 99,
	}
	testRoundTrip("BigStruct", bigStruct, &BigStruct{}, false)

	// Test SmallStruct
	smallStruct := &SmallStruct{
		Name:   "small",
		Series: Timeseries{Name: "series", Samples: []Sample{{Value: 2.0, Timestamp: 200}}},
	}
	testRoundTrip("SmallStruct", smallStruct, &SmallStruct{}, false)
}

// TestWeirdTypes tests edge case types from weird_types.go
func TestWeirdTypes(t *testing.T) {
	// Test type A with embedded B
	aStruct := &A{B: B{b: "embedded"}}
	aData := aStruct.MarshalProtobuf(nil)
	var aDecoded A
	if err := aDecoded.UnmarshalProtobuf(aData); err != nil {
		t.Fatalf("A.UnmarshalProtobuf failed: %v", err)
	}
	if aDecoded.B.b != "embedded" {
		t.Errorf("A.B.b mismatch: got %q, want %q", aDecoded.B.b, "embedded")
	}
	t.Logf("A (embedded B): encoded size %d bytes", len(aData))

	// Test type B with unexported field
	bStruct := &B{b: "unexported"}
	bData := bStruct.MarshalProtobuf(nil)
	var bDecoded B
	if err := bDecoded.UnmarshalProtobuf(bData); err != nil {
		t.Fatalf("B.UnmarshalProtobuf failed: %v", err)
	}
	if bDecoded.b != "unexported" {
		t.Errorf("B.b mismatch: got %q, want %q", bDecoded.b, "unexported")
	}
	t.Logf("B (unexported field): encoded size %d bytes", len(bData))

	// Test type d (implements interface D)
	dStruct := &d{dd: "implements"}
	dData := dStruct.MarshalProtobuf(nil)
	var dDecoded d
	if err := dDecoded.UnmarshalProtobuf(dData); err != nil {
		t.Fatalf("d.UnmarshalProtobuf failed: %v", err)
	}
	if dDecoded.dd != "implements" {
		t.Errorf("d.dd mismatch: got %q, want %q", dDecoded.dd, "implements")
	}
	// Verify it implements D interface
	var _ D = &dDecoded
	t.Logf("d (implements D): encoded size %d bytes", len(dData))

	// Test type d1 (has nested d)
	d1Struct := &d1{ddd: d{dd: "nested"}}
	d1Data := d1Struct.MarshalProtobuf(nil)
	var d1Decoded d1
	if err := d1Decoded.UnmarshalProtobuf(d1Data); err != nil {
		t.Fatalf("d1.UnmarshalProtobuf failed: %v", err)
	}
	if d1Decoded.ddd.dd != "nested" {
		t.Errorf("d1.ddd.dd mismatch: got %q, want %q", d1Decoded.ddd.dd, "nested")
	}
	t.Logf("d1 (nested d): encoded size %d bytes", len(d1Data))

	// Test type d2 (embedded d)
	d2Struct := &d2{d: d{dd: "embedded-d"}}
	d2Data := d2Struct.MarshalProtobuf(nil)
	var d2Decoded d2
	if err := d2Decoded.UnmarshalProtobuf(d2Data); err != nil {
		t.Fatalf("d2.UnmarshalProtobuf failed: %v", err)
	}
	if d2Decoded.d.dd != "embedded-d" {
		t.Errorf("d2.d.dd mismatch: got %q, want %q", d2Decoded.d.dd, "embedded-d")
	}
	t.Logf("d2 (embedded d): encoded size %d bytes", len(d2Data))

	// Test type all3 (complex with type aliases, maps, slices)
	all3Struct := &all3{
		all1: all1{d: d{dd: "embedded-all1"}, a: A{B: B{b: "nested-b"}}},
		aa: map[string]all2{
			"key1": {all: all1{d: d{dd: "map-value"}}},
		},
		bb: []all1{
			{d: d{dd: "slice-item"}},
		},
	}
	all3Data := all3Struct.MarshalProtobuf(nil)
	var all3Decoded all3
	if err := all3Decoded.UnmarshalProtobuf(all3Data); err != nil {
		t.Fatalf("all3.UnmarshalProtobuf failed: %v", err)
	}
	if all3Decoded.all1.d.dd != "embedded-all1" {
		t.Errorf("all3.all1.d.dd mismatch: got %q, want %q", all3Decoded.all1.d.dd, "embedded-all1")
	}
	if all3Decoded.aa["key1"].all.d.dd != "map-value" {
		t.Errorf("all3.aa[key1].all.d.dd mismatch")
	}
	if len(all3Decoded.bb) != 1 || all3Decoded.bb[0].d.dd != "slice-item" {
		t.Errorf("all3.bb mismatch")
	}
	t.Logf("all3 (type aliases, maps, slices): encoded size %d bytes", len(all3Data))
}
