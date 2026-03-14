package ir

import "testing"

func TestTypeAlignAndSizeForScalars(t *testing.T) {
	tests := []struct {
		name  string
		typ   Type
		align int
		size  int
	}{
		{name: "u8", typ: Type{Kind: TypeU8}, align: 1, size: 1},
		{name: "u16", typ: Type{Kind: TypeU16}, align: 2, size: 2},
		{name: "u32", typ: Type{Kind: TypeU32}, align: 4, size: 4},
		{name: "u64", typ: Type{Kind: TypeU64}, align: 8, size: 8},
		{name: "bool", typ: Type{Kind: TypeBool}, align: 1, size: 1},
		{name: "f32", typ: Type{Kind: TypeFloat32}, align: 4, size: 4},
		{name: "f64", typ: Type{Kind: TypeFloat64}, align: 8, size: 8},
		{name: "ptr", typ: PtrType(Type{Kind: TypeU8}), align: 8, size: 8},
	}

	for _, tt := range tests {
		if got := TypeAlign(tt.typ); got != tt.align {
			t.Fatalf("%s align: got %d, want %d", tt.name, got, tt.align)
		}

		if got := TypeSize(tt.typ); got != tt.size {
			t.Fatalf("%s size: got %d, want %d", tt.name, got, tt.size)
		}
	}
}

func TestStringLayout(t *testing.T) {
	typ := StringType()

	if got := TypeAlign(typ); got != 8 {
		t.Fatalf("string align: got %d, want 8", got)
	}

	if got := TypeSize(typ); got != 16 {
		t.Fatalf("string size: got %d, want 16", got)
	}

	if got := FieldOffset(typ, 0); got != 0 {
		t.Fatalf("string data offset: got %d, want 0", got)
	}

	if got := FieldOffset(typ, 1); got != 8 {
		t.Fatalf("string len offset: got %d, want 8", got)
	}
}

func TestStructLayoutWithPadding(t *testing.T) {
	typ := Type{
		Kind: TypeStruct,
		Name: "padded",
		Fields: []Field{
			{Name: "a", Type: Type{Kind: TypeU8}},
			{Name: "b", Type: Type{Kind: TypeU64}},
			{Name: "c", Type: Type{Kind: TypeU16}},
		},
	}

	if got := TypeAlign(typ); got != 8 {
		t.Fatalf("struct align: got %d, want 8", got)
	}

	if got := FieldOffset(typ, 0); got != 0 {
		t.Fatalf("field a offset: got %d, want 0", got)
	}

	if got := FieldOffset(typ, 1); got != 8 {
		t.Fatalf("field b offset: got %d, want 8", got)
	}

	if got := FieldOffset(typ, 2); got != 16 {
		t.Fatalf("field c offset: got %d, want 16", got)
	}

	if got := TypeSize(typ); got != 24 {
		t.Fatalf("struct size: got %d, want 24", got)
	}
}

func TestArrayLayoutUsesAlignedStride(t *testing.T) {
	elem := Type{
		Kind: TypeStruct,
		Name: "elem",
		Fields: []Field{
			{Name: "a", Type: Type{Kind: TypeU8}},
			{Name: "b", Type: Type{Kind: TypeU32}},
		},
	}

	array := Type{
		Kind: TypeArray,
		Elem: &elem,
		Len:  3,
	}

	if got := TypeAlign(elem); got != 4 {
		t.Fatalf("elem align: got %d, want 4", got)
	}

	if got := TypeSize(elem); got != 8 {
		t.Fatalf("elem size: got %d, want 8", got)
	}

	if got := TypeAlign(array); got != 4 {
		t.Fatalf("array align: got %d, want 4", got)
	}

	if got := TypeSize(array); got != 24 {
		t.Fatalf("array size: got %d, want 24", got)
	}
}
